CREATE STREAM truck_drivers_raw (
  truckId VARCHAR,
  carrierId VARCHAR,
  driver STRUCT<name VARCHAR, license VARCHAR>,
  updatedAt BIGINT
)
WITH (
  KAFKA_TOPIC = 'mongo.trackgo.truck-drivers',
  VALUE_FORMAT = 'JSON'
);

CREATE STREAM truck_drivers
WITH (KAFKA_TOPIC = 'truck-drivers-keyed', VALUE_FORMAT = 'JSON')
AS SELECT *
FROM truck_drivers_raw
PARTITION BY truckId;