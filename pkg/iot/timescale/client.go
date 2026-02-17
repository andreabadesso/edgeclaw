// Package timescale provides a TimescaleDB client for querying IoT sensor data.
//
// Each EdgeClaw edge node runs a local TimescaleDB instance populated by
// external ingestion processes (MQTT gateways, Telegraf, sensor scripts).
// This client provides the connection pool and query interface used by
// PicoClaw tools to read time-series data during heartbeat analysis cycles.
package timescale

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds TimescaleDB connection parameters.
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
	MaxConns int    `json:"max_conns"`
}

// Defaults fills in zero-valued fields with sensible defaults.
func (c *Config) Defaults() {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 5432
	}
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	if c.MaxConns == 0 {
		c.MaxConns = 5
	}
}

// Client wraps a pgx connection pool for TimescaleDB access.
type Client struct {
	pool *pgxpool.Pool
}

// New creates a TimescaleDB client and verifies the connection.
func New(ctx context.Context, cfg Config) (*Client, error) {
	cfg.Defaults()

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&pool_max_conns=%d",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode, cfg.MaxConns,
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("connect to timescaledb: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping timescaledb: %w", err)
	}

	log.Printf("[timescale] connected to %s:%d/%s", cfg.Host, cfg.Port, cfg.DBName)
	return &Client{pool: pool}, nil
}

// Close shuts down the connection pool.
func (c *Client) Close() {
	c.pool.Close()
}

// QueryJSON executes a read-only SQL query and returns the results as a
// JSON-encoded string. This is the primary interface used by the
// query_timescaledb PicoClaw tool.
func (c *Client) QueryJSON(ctx context.Context, query string, args ...any) (string, error) {
	rows, err := c.pool.Query(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	fieldDescs := rows.FieldDescriptions()
	var results []map[string]any

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return "", fmt.Errorf("scan row: %w", err)
		}
		row := make(map[string]any, len(fieldDescs))
		for i, fd := range fieldDescs {
			row[fd.Name] = values[i]
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate rows: %w", err)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal results: %w", err)
	}

	return string(data), nil
}

// EnsureSchema creates the default sensor hypertable if it doesn't exist.
func (c *Client) EnsureSchema(ctx context.Context) error {
	schema := `
		CREATE TABLE IF NOT EXISTS sensors (
			time        TIMESTAMPTZ NOT NULL,
			sensor_type TEXT NOT NULL,
			value       DOUBLE PRECISION,
			device_id   TEXT
		);
		SELECT create_hypertable('sensors', 'time', if_not_exists => TRUE);
	`

	_, err := c.pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("ensure schema: %w", err)
	}

	log.Println("[timescale] sensor hypertable schema ensured")
	return nil
}
