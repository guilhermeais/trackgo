FROM confluentinc/cp-kafka-connect:7.5.0

RUN confluent-hub install --no-prompt mongodb/kafka-connect-mongodb:latest
