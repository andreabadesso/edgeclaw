-- seed.sql — Insert realistic demo data into the EdgeClaw sensors hypertable.
--
-- This script populates ~100 rows of sensor readings spanning the last 24 hours
-- across five sensor types, each with 2-3 device IDs.  Several deliberate
-- anomalies are included so that the iot-monitor skill and query_timescaledb
-- tool have interesting patterns to detect.
--
-- Anomalies planted:
--   1. Energy spike (~3x normal) around 6 hours ago           (meter-01)
--   2. Water tank level dropping below 20% in the last hour   (tank-01)
--   3. Gradual pressure decline over the last 4 hours         (psi-01)
--   4. Water usage spike coinciding with the pressure drop    (flow-02)
--
-- Usage:
--   psql -U edgeclaw -d edgeclaw -f scripts/seed.sql

BEGIN;

-- ============================================================================
-- 1. ENERGY  (sensor_type = 'energy')
--    Normal range: 1.0 - 3.5 kW
--    Devices: meter-01, meter-02
-- ============================================================================

-- meter-01: normal readings over 24 h
INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '24 hours',  'energy', 1.2, 'meter-01'),
  (NOW() - INTERVAL '22 hours',  'energy', 1.1, 'meter-01'),
  (NOW() - INTERVAL '20 hours',  'energy', 1.4, 'meter-01'),
  (NOW() - INTERVAL '16 hours',  'energy', 2.8, 'meter-01'),
  (NOW() - INTERVAL '12 hours',  'energy', 2.9, 'meter-01'),
  (NOW() - INTERVAL '10 hours',  'energy', 2.5, 'meter-01'),
  (NOW() - INTERVAL '8 hours',   'energy', 2.3, 'meter-01'),

  -- ANOMALY: energy spike ~6 hours ago (~3x normal for this time of day)
  (NOW() - INTERVAL '6 hours 10 minutes', 'energy', 8.7, 'meter-01'),   -- << ANOMALY: spike start
  (NOW() - INTERVAL '6 hours',            'energy', 9.2, 'meter-01'),   -- << ANOMALY: spike peak
  (NOW() - INTERVAL '5 hours 50 minutes', 'energy', 7.8, 'meter-01'),   -- << ANOMALY: spike tail

  (NOW() - INTERVAL '5 hours',   'energy', 2.6, 'meter-01'),
  (NOW() - INTERVAL '4 hours',   'energy', 2.4, 'meter-01'),
  (NOW() - INTERVAL '2 hours',   'energy', 1.8, 'meter-01'),
  (NOW() - INTERVAL '1 hour',    'energy', 1.5, 'meter-01'),
  (NOW() - INTERVAL '30 minutes','energy', 1.3, 'meter-01');

-- meter-02: normal readings (different building / circuit)
INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '24 hours',  'energy', 0.8, 'meter-02'),
  (NOW() - INTERVAL '20 hours',  'energy', 0.9, 'meter-02'),
  (NOW() - INTERVAL '16 hours',  'energy', 1.5, 'meter-02'),
  (NOW() - INTERVAL '12 hours',  'energy', 1.8, 'meter-02'),
  (NOW() - INTERVAL '8 hours',   'energy', 1.6, 'meter-02'),
  (NOW() - INTERVAL '4 hours',   'energy', 1.2, 'meter-02'),
  (NOW() - INTERVAL '1 hour',    'energy', 0.9, 'meter-02');

-- ============================================================================
-- 2. WATER USAGE  (sensor_type = 'water_usage')
--    Normal range: 0.5 - 5.0 L/min
--    Devices: flow-01, flow-02
-- ============================================================================

-- flow-01: normal
INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '24 hours',  'water_usage', 0.8, 'flow-01'),
  (NOW() - INTERVAL '20 hours',  'water_usage', 1.2, 'flow-01'),
  (NOW() - INTERVAL '16 hours',  'water_usage', 3.0, 'flow-01'),
  (NOW() - INTERVAL '12 hours',  'water_usage', 4.1, 'flow-01'),
  (NOW() - INTERVAL '8 hours',   'water_usage', 2.2, 'flow-01'),
  (NOW() - INTERVAL '4 hours',   'water_usage', 1.5, 'flow-01'),
  (NOW() - INTERVAL '1 hour',    'water_usage', 0.7, 'flow-01');

-- flow-02: includes ANOMALY — usage spike during the last 4 hours,
-- coinciding with the pressure drop on psi-01 (correlation signal).
INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '24 hours',  'water_usage', 0.5, 'flow-02'),
  (NOW() - INTERVAL '20 hours',  'water_usage', 0.7, 'flow-02'),
  (NOW() - INTERVAL '16 hours',  'water_usage', 1.1, 'flow-02'),
  (NOW() - INTERVAL '12 hours',  'water_usage', 1.4, 'flow-02'),
  (NOW() - INTERVAL '8 hours',   'water_usage', 1.0, 'flow-02'),
  (NOW() - INTERVAL '6 hours',   'water_usage', 1.2, 'flow-02'),

  -- ANOMALY: water usage spike — possible leak or runaway irrigation
  (NOW() - INTERVAL '4 hours',            'water_usage', 8.5,  'flow-02'),   -- << ANOMALY: spike start
  (NOW() - INTERVAL '3 hours 30 minutes', 'water_usage', 11.2, 'flow-02'),   -- << ANOMALY: peak
  (NOW() - INTERVAL '3 hours',            'water_usage', 10.8, 'flow-02'),   -- << ANOMALY
  (NOW() - INTERVAL '2 hours 30 minutes', 'water_usage', 9.7,  'flow-02'),   -- << ANOMALY
  (NOW() - INTERVAL '2 hours',            'water_usage', 7.3,  'flow-02'),   -- << ANOMALY: tapering
  (NOW() - INTERVAL '1 hour',             'water_usage', 3.1,  'flow-02'),   -- << ANOMALY: tail
  (NOW() - INTERVAL '30 minutes',         'water_usage', 1.5,  'flow-02');

-- ============================================================================
-- 3. PRESSURE  (sensor_type = 'pressure')
--    Normal range: 45 - 55 psi
--    Devices: psi-01, psi-02
-- ============================================================================

-- psi-02: stable reference sensor
INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '24 hours',  'pressure', 50.1, 'psi-02'),
  (NOW() - INTERVAL '20 hours',  'pressure', 50.4, 'psi-02'),
  (NOW() - INTERVAL '16 hours',  'pressure', 49.8, 'psi-02'),
  (NOW() - INTERVAL '12 hours',  'pressure', 50.2, 'psi-02'),
  (NOW() - INTERVAL '8 hours',   'pressure', 50.0, 'psi-02'),
  (NOW() - INTERVAL '4 hours',   'pressure', 49.9, 'psi-02'),
  (NOW() - INTERVAL '2 hours',   'pressure', 50.3, 'psi-02'),
  (NOW() - INTERVAL '1 hour',    'pressure', 50.1, 'psi-02');

-- psi-01: ANOMALY — gradual pressure decline over the last 4 hours.
-- This correlates with the water usage spike on flow-02 (possible leak).
INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '24 hours',  'pressure', 51.0, 'psi-01'),
  (NOW() - INTERVAL '20 hours',  'pressure', 50.8, 'psi-01'),
  (NOW() - INTERVAL '16 hours',  'pressure', 50.9, 'psi-01'),
  (NOW() - INTERVAL '12 hours',  'pressure', 50.3, 'psi-01'),
  (NOW() - INTERVAL '8 hours',   'pressure', 50.1, 'psi-01'),
  (NOW() - INTERVAL '6 hours',   'pressure', 49.8, 'psi-01'),

  -- ANOMALY: gradual decline begins — coincides with flow-02 water spike
  (NOW() - INTERVAL '4 hours',            'pressure', 47.5, 'psi-01'),   -- << ANOMALY: decline start
  (NOW() - INTERVAL '3 hours 30 minutes', 'pressure', 45.2, 'psi-01'),   -- << ANOMALY
  (NOW() - INTERVAL '3 hours',            'pressure', 43.1, 'psi-01'),   -- << ANOMALY
  (NOW() - INTERVAL '2 hours 30 minutes', 'pressure', 41.0, 'psi-01'),   -- << ANOMALY
  (NOW() - INTERVAL '2 hours',            'pressure', 39.5, 'psi-01'),   -- << ANOMALY
  (NOW() - INTERVAL '1 hour 30 minutes',  'pressure', 38.2, 'psi-01'),   -- << ANOMALY
  (NOW() - INTERVAL '1 hour',             'pressure', 37.8, 'psi-01'),   -- << ANOMALY: below safe range
  (NOW() - INTERVAL '30 minutes',         'pressure', 37.1, 'psi-01');   -- << ANOMALY: still declining

-- ============================================================================
-- 4. WATER TANK LEVEL  (sensor_type = 'water_tank_level')
--    Normal range: 40% - 95%
--    Critical threshold: < 20%
--    Devices: tank-01, tank-02
-- ============================================================================

-- tank-02: healthy tank
INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '24 hours',  'water_tank_level', 88.0, 'tank-02'),
  (NOW() - INTERVAL '20 hours',  'water_tank_level', 85.5, 'tank-02'),
  (NOW() - INTERVAL '16 hours',  'water_tank_level', 82.0, 'tank-02'),
  (NOW() - INTERVAL '12 hours',  'water_tank_level', 78.5, 'tank-02'),
  (NOW() - INTERVAL '8 hours',   'water_tank_level', 91.0, 'tank-02'),
  (NOW() - INTERVAL '4 hours',   'water_tank_level', 87.0, 'tank-02'),
  (NOW() - INTERVAL '2 hours',   'water_tank_level', 84.0, 'tank-02'),
  (NOW() - INTERVAL '1 hour',    'water_tank_level', 82.5, 'tank-02');

-- tank-01: ANOMALY — level is draining and recently crossed the 20% critical threshold.
-- Likely related to the water usage anomaly on flow-02.
INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '24 hours',  'water_tank_level', 72.0, 'tank-01'),
  (NOW() - INTERVAL '20 hours',  'water_tank_level', 65.0, 'tank-01'),
  (NOW() - INTERVAL '16 hours',  'water_tank_level', 56.0, 'tank-01'),
  (NOW() - INTERVAL '12 hours',  'water_tank_level', 48.0, 'tank-01'),
  (NOW() - INTERVAL '8 hours',   'water_tank_level', 38.5, 'tank-01'),
  (NOW() - INTERVAL '6 hours',   'water_tank_level', 33.0, 'tank-01'),

  -- ANOMALY: tank level enters dangerous territory
  (NOW() - INTERVAL '4 hours',   'water_tank_level', 26.0, 'tank-01'),   -- << ANOMALY: low
  (NOW() - INTERVAL '3 hours',   'water_tank_level', 22.0, 'tank-01'),   -- << ANOMALY: approaching critical
  (NOW() - INTERVAL '2 hours',   'water_tank_level', 18.5, 'tank-01'),   -- << ANOMALY: BELOW 20% CRITICAL
  (NOW() - INTERVAL '1 hour',    'water_tank_level', 15.0, 'tank-01'),   -- << ANOMALY: CRITICAL
  (NOW() - INTERVAL '30 minutes','water_tank_level', 12.8, 'tank-01');   -- << ANOMALY: CRITICAL — action needed

-- ============================================================================
-- 5. TEMPERATURE  (sensor_type = 'temperature')
--    Normal range: 18 - 28 C (indoor equipment / server closet)
--    Devices: therm-01, therm-02, therm-03
-- ============================================================================

INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '24 hours',  'temperature', 21.0, 'therm-01'),
  (NOW() - INTERVAL '20 hours',  'temperature', 20.5, 'therm-01'),
  (NOW() - INTERVAL '16 hours',  'temperature', 22.0, 'therm-01'),
  (NOW() - INTERVAL '12 hours',  'temperature', 23.5, 'therm-01'),
  (NOW() - INTERVAL '8 hours',   'temperature', 24.0, 'therm-01'),
  (NOW() - INTERVAL '4 hours',   'temperature', 23.0, 'therm-01'),
  (NOW() - INTERVAL '2 hours',   'temperature', 22.5, 'therm-01'),
  (NOW() - INTERVAL '1 hour',    'temperature', 22.0, 'therm-01');

INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '24 hours',  'temperature', 19.5, 'therm-02'),
  (NOW() - INTERVAL '18 hours',  'temperature', 19.0, 'therm-02'),
  (NOW() - INTERVAL '12 hours',  'temperature', 20.5, 'therm-02'),
  (NOW() - INTERVAL '8 hours',   'temperature', 21.0, 'therm-02'),
  (NOW() - INTERVAL '4 hours',   'temperature', 20.0, 'therm-02'),
  (NOW() - INTERVAL '1 hour',    'temperature', 19.5, 'therm-02');

INSERT INTO sensors (time, sensor_type, value, device_id) VALUES
  (NOW() - INTERVAL '24 hours',  'temperature', 25.0, 'therm-03'),
  (NOW() - INTERVAL '18 hours',  'temperature', 24.5, 'therm-03'),
  (NOW() - INTERVAL '12 hours',  'temperature', 26.0, 'therm-03'),
  (NOW() - INTERVAL '8 hours',   'temperature', 27.0, 'therm-03'),
  (NOW() - INTERVAL '4 hours',   'temperature', 26.5, 'therm-03'),
  (NOW() - INTERVAL '1 hour',    'temperature', 25.5, 'therm-03');

COMMIT;

-- ============================================================================
-- Summary: verify the seed data
-- ============================================================================
SELECT
    sensor_type,
    COUNT(*)                    AS row_count,
    COUNT(DISTINCT device_id)   AS devices,
    ROUND(MIN(value)::numeric, 1)  AS min_val,
    ROUND(AVG(value)::numeric, 1)  AS avg_val,
    ROUND(MAX(value)::numeric, 1)  AS max_val
FROM sensors
WHERE time > NOW() - INTERVAL '25 hours'
GROUP BY sensor_type
ORDER BY sensor_type;
