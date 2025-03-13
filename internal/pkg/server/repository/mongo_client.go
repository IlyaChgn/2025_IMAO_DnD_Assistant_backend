package repository

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"log"
)

func NewMongoConnectionURI(user, password, host, port string) string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%s", user, password, host, port)
}

func ConnectToMongoDatabase(ctx context.Context, uri string, dnName string) *mongo.Database {
	client := newMongoClient(ctx, uri)

	return client.Database(dnName)
}

func newMongoClient(ctx context.Context, uri string) *mongo.Client {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Cannot connect to MongoDB client %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Cannot ping MongoDB %v", err)
	}

	return client
}
