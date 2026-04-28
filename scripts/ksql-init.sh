#!/bin/sh
set -e

wait_for() {
  name="$1"
  shift
  while ! "$@" >/dev/null 2>&1; do
    echo "Waiting for $name..."
    sleep 5
  done
}

wait_for "Kafka Connect" curl -fsS http://kafka-connect:8083/
wait_for "ksqldb server" curl -fsS http://ksqldb-server:8088/info

if curl -fsS http://kafka-connect:8083/connectors/mongo-source-truck-drivers >/dev/null 2>&1; then
  echo "Dropping existing MongoDB source connector..."
  curl -X DELETE http://kafka-connect:8083/connectors/mongo-source-truck-drivers
fi

echo "Registering MongoDB source connector..."
curl -X POST -H "Content-Type: application/json" --data @/connectors/mongo-source.json http://kafka-connect:8083/connectors

echo "Creating KSQL stream/table..."
curl -X POST \
  -H "Content-Type: application/vnd.ksql.v1+json; charset=utf-8" \
  --data "$(jq -n --rawfile sql /ksql/schemas.sql '{ksql: $sql, streamsProperties: {}}')" \
  http://ksqldb-server:8088/ksql

echo "KSQL init completed."