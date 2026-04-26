package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type seedConfig struct {
	NumberOfCarriers int
	TrucksPerCarrier int
	MongoURI         string
	Database         string
	Collection       string
}

type driver struct {
	Name    string `bson:"name"`
	License string `bson:"license"`
}

type truckDriver struct {
	TruckID   string    `bson:"truckId"`
	CarrierID string    `bson:"carrierId"`
	Driver    driver    `bson:"driver"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

var driverNames = []string{
	"Ana Silva",
	"Bruno Costa",
	"Carla Mendes",
	"Diego Alves",
	"Eduarda Faria",
	"Felipe Rocha",
	"Gabi Souza",
	"Hugo Lima",
	"Isabela Nunes",
	"João Pereira",
	"Laura Cardoso",
	"Marcelo Pinto",
}

var licensePrefixes = []string{"BR", "SP", "RJ", "MG", "ES"}

func main() {
	_ = godotenv.Load()

	cfg := loadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		fmt.Printf("failed to connect to MongoDB: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	if err := client.Ping(ctx, nil); err != nil {
		fmt.Printf("failed to ping MongoDB: %v\n", err)
		os.Exit(1)
	}

	collection := client.Database(cfg.Database).Collection(cfg.Collection)

	if _, err := collection.DeleteMany(ctx, bson.M{}); err != nil {
		fmt.Printf("failed to clear existing collection: %v\n", err)
		os.Exit(1)
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	docs := make([]interface{}, 0, cfg.NumberOfCarriers*cfg.TrucksPerCarrier)

	for carrierId := 1; carrierId <= cfg.NumberOfCarriers; carrierId++ {
		for truckId := 1; truckId <= cfg.TrucksPerCarrier; truckId++ {
			driverInfo := driver{
				Name:    randomDriverName(),
				License: randomLicense(),
			}

			docs = append(docs, truckDriver{
				TruckID:   fmt.Sprintf("Carrier%d-Truck%d", carrierId, truckId),
				CarrierID: fmt.Sprintf("Carrier%d", carrierId),
				Driver:    driverInfo,
				UpdatedAt: time.Now(),
			})
		}
	}

	result, err := collection.InsertMany(ctx, docs)
	if err != nil {
		fmt.Printf("failed to seed MongoDB collection: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Inserted %d driver records into %s.%s\n", len(result.InsertedIDs), cfg.Database, cfg.Collection)
}

func loadConfig() seedConfig {
	return seedConfig{
		NumberOfCarriers: getEnvInt("NUMBER_OF_CARRIERS", 5),
		TrucksPerCarrier: getEnvInt("TRUCKS_PER_CARRIER", 10),
		MongoURI:         getEnvString("MONGO_URI", "mongodb://localhost:27017"),
		Database:         getEnvString("MONGO_DB", "trackgo"),
		Collection:       getEnvString("MONGO_COLLECTION", "truck-drivers"),
	}
}

func randomDriverName() string {
	return driverNames[rand.Intn(len(driverNames))]
}

func randomLicense() string {
	return fmt.Sprintf("%s-%06d", licensePrefixes[rand.Intn(len(licensePrefixes))], rand.Intn(1000000))
}

func getEnvString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
