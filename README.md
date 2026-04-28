# TrackGo - GPS Truck Simulator

A Go application that simulates GPS data for truck fleets moving between Brazilian cities, publishing data to Apache Kafka.

## Quick Start with Docker

**Option 1: Manual setup**
```bash
docker-compose up -d
# Wait 30 seconds for Kafka to start
go run cmd/main.go
```

**Option 2: Automated setup**
```bash
# Linux/Mac
./setup.sh

# Windows
setup.bat
```

**Monitor and manage:**
- Kafka UI: http://localhost:8080
- View logs: `docker-compose logs -f kafka`
- Stop: `docker-compose down`

## Docker Services

The `docker-compose.yml` includes:

- **Zookeeper**: Required for Kafka coordination
- **Kafka**: Message broker running on `localhost:9092`
- **Kafka UI**: Web interface for monitoring topics and messages at http://localhost:8080
- **MongoDB**: Document store running on `localhost:27017`
- **Mongo Express**: MongoDB admin UI at http://localhost:8081

## Features

- Simulates multiple carriers with configurable number of trucks per carrier
- Realistic GPS routes between major Brazilian cities (São Paulo, Rio de Janeiro, Curitiba)
- Random route assignment for varied simulation
- Configurable Kafka publishing
- Environment-based configuration
- Docker Compose setup for easy Kafka deployment

## Configuration

The application supports configuration through environment variables. You can set these directly or use a `.env` file.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NUMBER_OF_CARRIERS` | 5 | Number of carriers to simulate |
| `TRUCKS_PER_CARRIER` | 10 | Number of trucks per carrier |
| `KAFKA_SERVERS` | localhost:9092 | Kafka bootstrap servers |
| `KAFKA_TOPIC` | raw-gps-data | Topic to publish GPS data |
| `MONGO_URI` | mongodb://localhost:27017 | MongoDB connection URI |
| `MONGO_DB` | trackgo | MongoDB database name |
| `MONGO_COLLECTION` | truck-drivers | MongoDB collection name |

### Using .env File

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your desired values:
   ```env
   NUMBER_OF_CARRIERS=3
   TRUCKS_PER_CARRIER=5
   KAFKA_SERVERS=kafka-cluster:9092
   KAFKA_TOPIC=production-gps
   ```

3. Run the application:
   ```bash
   go run cmd/main.go
   ```

### Command Line Examples

```bash
# Use defaults
go run cmd/main.go

# Override with environment variables
NUMBER_OF_CARRIERS=10 KAFKA_SERVERS=prod-kafka:9092 go run cmd/main.go

# Use .env file
go run cmd/main.go
```

## Routes

The simulator includes predefined routes between:
- São Paulo ↔ Rio de Janeiro
- São Paulo → Curitiba
- Rio de Janeiro → Curitiba
- Curitiba → São Paulo

Each truck is randomly assigned one of these routes at startup.

## MongoDB and Seeder

This project now includes a MongoDB seeder for the `truck-drivers` collection.

- Service: `mongodb` on `mongodb://localhost:27017`
- Database: `trackgo`
- Collection: `truck-drivers`
- Document fields: `truckId`, `carrierId`, `driver.name`, `driver.license`, `updatedAt`

### Seed MongoDB

1. Start Docker services:
   ```bash
docker-compose up -d
```
2. Run the seeder:
   ```bash
go run ./cmd/seed
```

### MongoDB CDC to Kafka Connect

The `kafka-connect` service exposes Kafka Connect on `http://localhost:8083`.
This repository now also includes a `ksqldb-server` service that automatically registers the MongoDB source connector and creates a KSQL table from the CDC topic on startup.

Start everything with:

```bash
docker-compose up -d --build
```

The init service will:
- register `connectors/mongo-source.json` with Kafka Connect if it is not already present
- create a `truck_drivers_source` stream and `drivers_table` table in ksqlDB

The connector uses `copy.existing=true` so existing `truck-drivers` documents are copied into Kafka before CDC begins.
The source topic prefix is `mongo`, producing events under topics like `mongo.trackgo.truck-drivers`.

ksqlDB is available at:
- `http://localhost:8088`

### MongoDB UI

- Mongo Express: http://localhost:8081

The seeder uses `NUMBER_OF_CARRIERS` and `TRUCKS_PER_CARRIER` from `.env` or environment variables to generate matching records.

### MongoDB UI

- Mongo Express: http://localhost:8081

## Output

The application publishes GPS data in JSON format to the configured Kafka topic:

```json
{
  "truckId": "Truck1",
  "carrierId": "Carrier1",
  "latitude": -23.5505,
  "longitude": -46.6333,
  "date": "2024-01-15T10:30:00Z"
}
```

## Troubleshooting

### Kafka Connection Issues
- Ensure Docker containers are running: `docker-compose ps`
- Check Kafka logs: `docker-compose logs kafka`
- Verify port availability: `netstat -an | find "9092"`

### Permission Issues on Windows
If you encounter permission errors with Docker:
```bash
# Reset Docker containers
docker-compose down -v
docker-compose up --build
```

### Kafka UI Not Accessible
- Wait for health checks to pass: `docker-compose ps`
- Check UI logs: `docker-compose logs kafka-ui`

## Dependencies

- Go 1.19+
- Docker and Docker Compose
- Apache Kafka (provided via Docker)
- [confluent-kafka-go](https://github.com/confluentinc/confluent-kafka-go)
- [godotenv](https://github.com/joho/godotenv)

## Building

```bash
go mod tidy
go build ./cmd
```

## Running

Make sure Kafka is running and accessible, then:

```bash
./track-go
```

Or with custom configuration:

```bash
KAFKA_SERVERS=localhost:9092,localhost:9093 ./track-go
```