package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	uri := os.Getenv("MONGODB")
	if uri == "" {
		uri = "mongodb://encounterium_root_user:HfMu8w79hPUEJyrS3RchS2Gs@encounterium.ru:27019/?authSource=admin&tls=true"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Connect error:", err)
	}
	defer client.Disconnect(ctx)

	coll := client.Database("bestiary_db").Collection("creatures")

	// Find all creatures
	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		log.Fatal("Find error:", err)
	}
	defer cursor.Close(ctx)

	var creatures []bson.M
	if err := cursor.All(ctx, &creatures); err != nil {
		log.Fatal("Cursor error:", err)
	}

	fmt.Printf("Found %d creatures\n", len(creatures))

	// Create backup directory
	backupDir := "backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		log.Fatal("Mkdir error:", err)
	}

	// Write to JSON file
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s/creatures_backup_%s.json", backupDir, timestamp)

	file, err := os.Create(filename)
	if err != nil {
		log.Fatal("Create file error:", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(creatures); err != nil {
		log.Fatal("Encode error:", err)
	}

	// Get file size
	info, _ := file.Stat()
	fmt.Printf("Backup saved to: %s\n", filename)
	fmt.Printf("File size: %.2f MB\n", float64(info.Size())/(1024*1024))
}
