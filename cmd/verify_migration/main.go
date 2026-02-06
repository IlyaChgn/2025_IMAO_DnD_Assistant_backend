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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Connect error:", err)
	}
	defer client.Disconnect(ctx)

	coll := client.Database("bestiary_db").Collection("creatures")

	// Stats
	withMovement, _ := coll.CountDocuments(ctx, bson.M{"movement": bson.M{"$exists": true}})
	withVision, _ := coll.CountDocuments(ctx, bson.M{"vision": bson.M{"$exists": true}})
	withActions, _ := coll.CountDocuments(ctx, bson.M{"structuredActions": bson.M{"$exists": true}})
	total, _ := coll.CountDocuments(ctx, bson.M{})

	fmt.Println("=== MIGRATION VERIFICATION ===")
	fmt.Printf("Total creatures: %d\n", total)
	fmt.Printf("With movement: %d\n", withMovement)
	fmt.Printf("With vision: %d\n", withVision)
	fmt.Printf("With structuredActions: %d\n", withActions)

	// Sample creature
	fmt.Println("\n=== SAMPLE: Гоблин ===")
	var goblin bson.M
	err = coll.FindOne(ctx, bson.M{"name.rus": "Гоблин"}).Decode(&goblin)
	if err != nil {
		log.Fatal("FindOne error:", err)
	}

	// Print only new fields
	sample := bson.M{
		"name":              goblin["name"],
		"movement":          goblin["movement"],
		"vision":            goblin["vision"],
		"structuredActions": goblin["structuredActions"],
	}

	jsonBytes, _ := json.MarshalIndent(sample, "", "  ")
	fmt.Println(string(jsonBytes))

	// Sample with darkvision + fly
	fmt.Println("\n=== SAMPLE: Псевдодракон ===")
	var pseudo bson.M
	err = coll.FindOne(ctx, bson.M{"name.rus": "Псевдодракон"}).Decode(&pseudo)
	if err == nil {
		sample2 := bson.M{
			"name":     pseudo["name"],
			"movement": pseudo["movement"],
			"vision":   pseudo["vision"],
		}
		jsonBytes2, _ := json.MarshalIndent(sample2, "", "  ")
		fmt.Println(string(jsonBytes2))
	}
}
