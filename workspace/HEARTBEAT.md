# IoT Monitoring Tasks

- Query latest sensor data from TimescaleDB (last 30 minutes) using the query_timescaledb tool. Include all sensor types.
- Analyze energy consumption readings for outliers using IQR or Z-score methods. Flag spikes that exceed 2 standard deviations from the rolling 24-hour average.
- Check water usage patterns for anomalies: sudden spikes may indicate burst pipes, sudden drops may indicate blockages or sensor failure.
- Monitor pressure levels against normal operating range. A pressure drop coinciding with an energy spike is a strong indicator of equipment fault -- correlate these signals.
- Check water tank levels against minimum thresholds. Levels below 20% require a critical alert. Levels below 40% require a warning.
- Compare all current readings against the sensors_hourly continuous aggregate for trend context. Note any sustained drift over the past 24 hours.
- If any anomalies are detected, use the fleet_alert tool to broadcast to other nodes. Use appropriate severity levels (info/warning/critical).
