#!/bin/bash

# TrackGo Development Setup Script

echo "🚀 Setting up TrackGo development environment..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if Docker Compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed."
    exit 1
fi

echo "📦 Starting Kafka infrastructure..."
docker-compose up -d

echo "⏳ Waiting for Kafka to be ready..."
sleep 30

# Check if Kafka is healthy
if docker-compose ps kafka | grep -q "healthy\|running"; then
    echo "✅ Kafka is running!"
    echo ""
    echo "🌐 Services available:"
    echo "  - Kafka: localhost:9092"
    echo "  - Kafka UI: http://localhost:8080"
    echo "  - Zookeeper: localhost:2181"
    echo ""
    echo "🚛 Run the simulator:"
    echo "  go run cmd/main.go"
    echo ""
    echo "🛑 To stop: docker-compose down"
else
    echo "❌ Kafka failed to start. Check logs:"
    echo "  docker-compose logs kafka"
    exit 1
fi