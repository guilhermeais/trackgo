DROP STREAM IF EXISTS enriched_gps_data DELETE TOPIC;
DROP TABLE IF EXISTS drivers_table DELETE TOPIC;
DROP STREAM IF EXISTS truck_drivers DELETE TOPIC;
DROP STREAM IF EXISTS truck_drivers_raw DELETE TOPIC;
DROP STREAM IF EXISTS raw_gps_data DELETE TOPIC;

SET 'auto.offset.reset' = 'earliest';

CREATE STREAM IF NOT EXISTS truck_drivers_raw (
  truckId VARCHAR,
  carrierId VARCHAR,
  driver STRUCT<name VARCHAR, license VARCHAR>
)
WITH (
  KAFKA_TOPIC = 'mongo.trackgo.truck-drivers',
  VALUE_FORMAT = 'JSON'
);

CREATE STREAM IF NOT EXISTS truck_drivers
WITH (KAFKA_TOPIC = 'truck-drivers-keyed', VALUE_FORMAT = 'JSON')
AS SELECT
  truckId,
  carrierId,
  driver
FROM truck_drivers_raw
PARTITION BY truckId;

CREATE TABLE IF NOT EXISTS drivers_table AS
  SELECT
    truckId as truckId,
    LATEST_BY_OFFSET(carrierId) AS carrierId,
    LATEST_BY_OFFSET(driver) AS driver
  FROM truck_drivers
  GROUP BY truckId
  EMIT CHANGES;

CREATE STREAM IF NOT EXISTS raw_gps_data (
  truckId VARCHAR,
  carrierId VARCHAR,
  latitude VARCHAR,
  longitude VARCHAR,
  date VARCHAR
)
WITH (
  KAFKA_TOPIC = 'raw-gps-data',
  VALUE_FORMAT = 'JSON'
);

CREATE STREAM IF NOT EXISTS enriched_gps_data
WITH (KAFKA_TOPIC = 'enriched-gps-data', VALUE_FORMAT = 'JSON')
AS SELECT 
  r.truckId as truckId,
  r.carrierId as carrierId,
  r.latitude as latitude,
  r.longitude as longitude,
  r.date as date,
  d.driver as driver
FROM raw_gps_data r
LEFT JOIN drivers_table d
  ON r.truckId = d.truckId
EMIT CHANGES;