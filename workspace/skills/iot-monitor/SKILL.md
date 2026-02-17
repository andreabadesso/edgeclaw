---
name: iot-monitor
description: IoT sensor monitoring and anomaly detection for EdgeClaw fleet nodes
---

# IoT Monitor Skill

You are an expert at analyzing IoT sensor time-series data for anomalies,
trends, and correlations. You have access to a local TimescaleDB instance
containing sensor readings and can query it using SQL.

## Sensor Data Schema

The `sensors` hypertable contains raw readings:
- `time` (TIMESTAMPTZ): When the reading was taken
- `sensor_type` (TEXT): Type of sensor (energy, water_usage, pressure, water_tank_level, temperature)
- `value` (DOUBLE PRECISION): The sensor reading
- `device_id` (TEXT): Which physical device produced this reading

The `sensors_hourly` continuous aggregate provides pre-computed rollups:
- `bucket` (TIMESTAMPTZ): Hour bucket
- `sensor_type`, `device_id`: Same as above
- `avg_value`, `min_value`, `max_value`: Aggregated values
- `sample_count`: Number of readings in the bucket

## Analysis Techniques

### Outlier Detection
Use IQR (Interquartile Range) for robust outlier detection:
```sql
SELECT sensor_type,
       percentile_cont(0.25) WITHIN GROUP (ORDER BY value) AS q1,
       percentile_cont(0.75) WITHIN GROUP (ORDER BY value) AS q3
FROM sensors
WHERE time > NOW() - INTERVAL '24 hours'
GROUP BY sensor_type;
```
Values outside Q1 - 1.5*IQR or Q3 + 1.5*IQR are outliers.

### Trend Analysis
Compare recent readings against historical averages:
```sql
SELECT sensor_type,
       AVG(value) AS current_avg,
       (SELECT AVG(avg_value) FROM sensors_hourly
        WHERE bucket > NOW() - INTERVAL '7 days'
        AND sensors_hourly.sensor_type = s.sensor_type) AS week_avg
FROM sensors s
WHERE time > NOW() - INTERVAL '30 minutes'
GROUP BY sensor_type;
```

### Cross-Sensor Correlation
Look for simultaneous anomalies across sensor types. An energy spike
coinciding with a pressure drop often indicates equipment malfunction.
Query both sensor types in the same time window and compare.

## Alert Thresholds

| Sensor Type | Warning | Critical |
|---|---|---|
| energy | >2 std dev above 24h avg | >3 std dev or sudden spike >50% |
| water_usage | >50% above 24h avg | >100% above avg or sudden spike |
| pressure | Trending down >10% over 6h | Drop >20% in 30 min |
| water_tank_level | Below 40% | Below 20% |
| temperature | >5C above normal range | >10C above or any below-freezing |
