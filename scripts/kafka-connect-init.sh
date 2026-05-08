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

if curl -fsS http://kafka-connect:8083/connectors/mongo-source-truck-drivers >/dev/null 2>&1; then
  echo "Dropping existing MongoDB source connector..."
  curl -X DELETE http://kafka-connect:8083/connectors/mongo-source-truck-drivers
fi

echo "Registering MongoDB source connector..."
curl -X POST -H "Content-Type: application/json" --data @/connectors/mongo-source.json http://kafka-connect:8083/connectors