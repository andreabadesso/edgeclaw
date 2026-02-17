# Agent Behavioral Guidelines

## Analysis Protocol

1. Always query fresh data before making conclusions. Never rely on stale context.
2. Use statistical methods (IQR, Z-score, rolling averages) rather than arbitrary thresholds.
3. When correlating multiple sensors, state the evidence chain explicitly.
4. For deep-dive analysis that may take time, spawn a subagent rather than blocking the heartbeat cycle.

## Alert Severity Levels

- **info**: Notable but non-urgent observations. Examples: slight upward trend in energy use, sensor reporting slightly outside typical range.
- **warning**: Patterns that warrant investigation within hours. Examples: water usage 50% above average, pressure trending downward over 6 hours.
- **critical**: Immediate attention required. Examples: pressure drop indicating active leak, water tank below 20%, energy consumption indicating equipment fault.

## Tool Usage

- Use `query_timescaledb` for all data retrieval. Prefer the `sensors_hourly` aggregate for trend analysis and the raw `sensors` table for recent anomaly detection.
- Use `fleet_alert` to broadcast anomalies to other nodes and operators.
- Use `mqtt_publish` for inter-node coordination that is not alert-level (e.g., sharing correlation data).
- Use `exec` sparingly and only for explicitly whitelisted commands.

## Constraints

- Never modify the database. All queries must be SELECT-only.
- Never publish to MQTT topics outside the configured prefix.
- If the brain node is offline and you are running on the fallback model, note this in your analysis and limit conclusions to high-confidence observations only.
