package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/joho/godotenv"
)

type Route struct {
	currentWaypoint int
	progressBetween float64
	waypoints       []GPSPoint
}

func (r *Route) GetNextPosition() GPSPoint {
	if r.currentWaypoint >= len(r.waypoints)-1 {
		// Reached the end, stay at last waypoint
		return r.waypoints[len(r.waypoints)-1]
	}

	current := r.waypoints[r.currentWaypoint]
	next := r.waypoints[r.currentWaypoint+1]

	lat := current.Latitude + (next.Latitude-current.Latitude)*r.progressBetween
	lon := current.Longitude + (next.Longitude-current.Longitude)*r.progressBetween

	r.progressBetween += 0.000694 // Approximately 1/1440 for 2-hour journey at 5s intervals

	if r.progressBetween >= 1.0 {
		r.currentWaypoint++
		r.progressBetween = 0.0
	}

	return GPSPoint{Latitude: lat, Longitude: lon}
}

type GPSPoint struct {
	Latitude  float64
	Longitude float64
}

type GPSPayload struct {
	TruckId   string    `json:"truckId"`
	CarrierId string    `json:"carrierId"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Date      time.Time `json:"date"`
}

const DEFAULT_NUMBER_OF_CARRIERS = 5
const DEFAULT_TRUCKS_PER_CARRIER = 10
const DEFAULT_KAFKA_SERVERS = "localhost:9092"
const DEFAULT_KAFKA_TOPIC = "raw-gps-data"

type Config struct {
	NumberOfCarriers int
	TrucksPerCarrier int
	KafkaServers     string
	KafkaTopic       string
}

func loadConfig() (Config, bool) {
	// Load .env file if it exists
	envLoaded := godotenv.Load() == nil

	return Config{
		NumberOfCarriers: getNumberOfCarriers(),
		TrucksPerCarrier: getTrucksPerCarrier(),
		KafkaServers:     getKafkaServers(),
		KafkaTopic:       getKafkaTopic(),
	}, envLoaded
}

// Predefined routes with waypoints
var routes = [][]GPSPoint{
	// Route 0: São Paulo to Rio de Janeiro
	{
		{-23.5505, -46.6333}, // São Paulo
		{-23.4000, -46.5000}, // Intermediate
		{-23.0000, -45.5000}, // Intermediate
		{-22.5000, -44.0000}, // Intermediate
		{-22.9068, -43.1729}, // Rio de Janeiro
	},
	// Route 1: Rio de Janeiro to São Paulo
	{
		{-22.9068, -43.1729}, // Rio de Janeiro
		{-22.5000, -44.0000}, // Intermediate
		{-23.0000, -45.5000}, // Intermediate
		{-23.4000, -46.5000}, // Intermediate
		{-23.5505, -46.6333}, // São Paulo
	},
	// Route 2: São Paulo to Curitiba
	{
		{-23.5505, -46.6333}, // São Paulo
		{-24.0000, -47.0000}, // Intermediate
		{-25.0000, -48.0000}, // Intermediate
		{-25.4284, -49.2671}, // Curitiba
	},
	// Route 3: Rio de Janeiro to Curitiba
	{
		{-22.9068, -43.1729}, // Rio de Janeiro
		{-23.5000, -44.0000}, // Intermediate
		{-24.5000, -46.0000}, // Intermediate
		{-25.0000, -48.0000}, // Intermediate
		{-25.4284, -49.2671}, // Curitiba
	},
	// Route 4: Curitiba to São Paulo
	{
		{-25.4284, -49.2671}, // Curitiba
		{-25.0000, -48.0000}, // Intermediate
		{-24.5000, -46.0000}, // Intermediate
		{-23.5000, -44.0000}, // Intermediate
		{-23.5505, -46.6333}, // São Paulo
	},
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	config, envLoaded := loadConfig()

	if envLoaded {
		fmt.Println("Configuration loaded from .env file")
	} else {
		fmt.Println("Configuration loaded from environment variables (no .env file found)")
	}

	fmt.Printf("Configuration values:\n")
	fmt.Printf("  Number of carriers: %d\n", config.NumberOfCarriers)
	fmt.Printf("  Trucks per carrier: %d\n", config.TrucksPerCarrier)
	fmt.Printf("  Kafka servers: %s\n", config.KafkaServers)
	fmt.Printf("  Kafka topic: %s\n", config.KafkaTopic)
	fmt.Println()

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": config.KafkaServers,
	})
	if err != nil {
		fmt.Printf("Failed to create Kafka producer: %v\n", err)
		os.Exit(1)
	}
	defer p.Close()

	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
				}
			}
		}
	}()

	topic := config.KafkaTopic

	numberOfCarriers := config.NumberOfCarriers
	truckPerCarrier := config.TrucksPerCarrier

	// Initialize the GPS data generator with the specified number of carriers and trucks per carrier
	generator := NewGPSDataGenerator(numberOfCarriers, truckPerCarrier)

	gpsChannel := make(chan GPSPayload)

	// Start the GPS data generator
	go generator.Start(ctx, gpsChannel)

	for gpsPayload := range gpsChannel {
		b, err := json.Marshal(gpsPayload)
		if err != nil {
			fmt.Printf("error sending message: %v\n", err)
			continue
		}

		err = p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          b,
			Key:            []byte(fmt.Sprintf("%s:%s", gpsPayload.CarrierId, gpsPayload.TruckId)),
		}, nil)
		if err != nil {
			fmt.Printf("erro producing kafka message: %v\n", err)
		}
	}
}

func NewGPSDataGenerator(numberOfCarriers, trucksPerCarrier int) *GPSDataGenerator {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	routesMap := make(map[string]*Route)
	for carrierId := 1; carrierId <= numberOfCarriers; carrierId++ {
		for truckId := 1; truckId <= trucksPerCarrier; truckId++ {
			key := fmt.Sprintf("Carrier%d-Truck%d", carrierId, truckId)
			routeIndex := rand.Intn(len(routes))
			routesMap[key] = &Route{
				waypoints: routes[routeIndex],
			}
		}
	}
	return &GPSDataGenerator{
		numberOfCarriers: numberOfCarriers,
		trucksPerCarrier: trucksPerCarrier,
		routes:           routesMap,
	}
}

type GPSDataGenerator struct {
	numberOfCarriers int
	trucksPerCarrier int
	routes           map[string]*Route
}

// Start begins the GPS data generation process. It will continue to generate data until the context is canceled.
// A carrier can have multiple trucks, and each truck will generate GPS data every 5 seconds.
// The GPS data includes interpolated latitude and longitude along predefined routes, along with the truck ID, carrier ID, and the timestamp of the data generation.
func (g *GPSDataGenerator) Start(ctx context.Context, gpsChannel chan<- GPSPayload) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(gpsChannel)
			return
		case <-ticker.C:
			for carrierId := 1; carrierId <= g.numberOfCarriers; carrierId++ {
				for truckId := 1; truckId <= g.trucksPerCarrier; truckId++ {
					routeKey := fmt.Sprintf("Carrier%d-Truck%d", carrierId, truckId)
					position := g.routes[routeKey].GetNextPosition()
					gpsData := GPSPayload{
						TruckId:   "Truck" + strconv.Itoa(truckId),
						CarrierId: "Carrier" + strconv.Itoa(carrierId),
						Latitude:  position.Latitude,
						Longitude: position.Longitude,
						Date:      time.Now(),
					}

					select {
					case gpsChannel <- gpsData:
						fmt.Printf("Generated GPS data for %s: %.6f, %.6f\n", routeKey, position.Latitude, position.Longitude)
					case <-ctx.Done():
						close(gpsChannel)
						return
					}
				}
			}
		}
	}

}

func getNumberOfCarriers() int {
	fromEnv := os.Getenv("NUMBER_OF_CARRIERS")
	numberOfCarriers, err := strconv.Atoi(fromEnv)
	if err != nil {
		return DEFAULT_NUMBER_OF_CARRIERS
	}
	return numberOfCarriers
}

func getTrucksPerCarrier() int {
	fromEnv := os.Getenv("TRUCKS_PER_CARRIER")
	trucksPerCarrier, err := strconv.Atoi(fromEnv)
	if err != nil {
		return DEFAULT_TRUCKS_PER_CARRIER
	}
	return trucksPerCarrier
}

func getKafkaServers() string {
	fromEnv := os.Getenv("KAFKA_SERVERS")
	if fromEnv == "" {
		return DEFAULT_KAFKA_SERVERS
	}
	return fromEnv
}

func getKafkaTopic() string {
	fromEnv := os.Getenv("KAFKA_TOPIC")
	if fromEnv == "" {
		return DEFAULT_KAFKA_TOPIC
	}
	return fromEnv
}
