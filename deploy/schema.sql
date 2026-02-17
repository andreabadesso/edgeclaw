-- EdgeClaw TimescaleDB Schema
-- Run this on each edge node's PostgreSQL/TimescaleDB instance.
-- The schema is applied automatically when using the Docker Compose stack
-- (mounted as /docker-entrypoint-initdb.d/01-schema.sql).

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Core sensor data hypertable.
CREATE TABLE IF NOT EXISTS sensors (
    time        TIMESTAMPTZ NOT NULL,
    sensor_type TEXT NOT NULL,
    value       DOUBLE PRECISION,
    device_id   TEXT
);

SELECT create_hypertable('sensors', 'time', if_not_exists => TRUE);

-- Enable compression for storage efficiency on edge devices (SD cards, small SSDs).
ALTER TABLE sensors SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'sensor_type'
);

-- Compress chunks older than 7 days automatically.
SELECT add_compression_policy('sensors', INTERVAL '7 days', if_not_exists => TRUE);

-- Retention: drop data older than 90 days to manage storage.
SELECT add_retention_policy('sensors', INTERVAL '90 days', if_not_exists => TRUE);

-- Indexes for common query patterns used by the query_timescaledb tool.
CREATE INDEX IF NOT EXISTS idx_sensors_type_time ON sensors (sensor_type, time DESC);
CREATE INDEX IF NOT EXISTS idx_sensors_device_time ON sensors (device_id, time DESC);

-- Continuous aggregate for hourly rollups (used by the iot-monitor skill for trend analysis).
CREATE MATERIALIZED VIEW IF NOT EXISTS sensors_hourly
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    sensor_type,
    device_id,
    AVG(value) AS avg_value,
    MIN(value) AS min_value,
    MAX(value) AS max_value,
    COUNT(*) AS sample_count
FROM sensors
GROUP BY bucket, sensor_type, device_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('sensors_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset   => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE
);
