// Package tools provides custom PicoClaw tools for IoT operations.
//
// These tools are registered into PicoClaw's ToolRegistry at startup,
// extending the agent with the ability to query sensor data, publish
// fleet messages, and broadcast alerts. Each tool follows PicoClaw's
// tool interface contract (Name, Description, Execute).
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/andreabadesso/edgeclaw/pkg/iot/timescale"
)

// QueryTimescaleDB allows PicoClaw to query the local TimescaleDB for sensor data.
// It accepts a read-only SQL query and returns results as JSON.
type QueryTimescaleDB struct {
	db *timescale.Client
}

// NewQueryTimescaleDB creates a new database query tool.
func NewQueryTimescaleDB(db *timescale.Client) *QueryTimescaleDB {
	return &QueryTimescaleDB{db: db}
}

func (t *QueryTimescaleDB) Name() string { return "query_timescaledb" }

func (t *QueryTimescaleDB) Description() string {
	return "Query the local TimescaleDB for IoT sensor data. " +
		"Provide a read-only SQL SELECT query. Returns results as JSON. " +
		"The 'sensors' hypertable has columns: time (TIMESTAMPTZ), sensor_type (TEXT), " +
		"value (DOUBLE PRECISION), device_id (TEXT). " +
		"A 'sensors_hourly' continuous aggregate provides pre-computed hourly rollups " +
		"with columns: bucket, sensor_type, device_id, avg_value, min_value, max_value, sample_count."
}

func (t *QueryTimescaleDB) Execute(ctx context.Context, args map[string]string) (string, error) {
	query, ok := args["query"]
	if !ok || query == "" {
		return "", fmt.Errorf("missing required argument: query")
	}

	normalized := strings.TrimSpace(strings.ToUpper(query))
	if !strings.HasPrefix(normalized, "SELECT") {
		return "", fmt.Errorf("only SELECT queries are allowed for safety")
	}

	return t.db.QueryJSON(ctx, query)
}
