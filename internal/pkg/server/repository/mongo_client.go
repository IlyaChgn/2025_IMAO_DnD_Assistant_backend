package repository

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewMongoConnectionURI(user, password, host, port string) string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%s/?tls=true", user, password, host, port)
}

func ConnectToMongoDatabase(ctx context.Context, uri string, dnName string) *mongo.Database {
	client := newMongoClient(ctx, uri)

	return client.Database(dnName)
}

func newMongoClient(ctx context.Context, uri string) *mongo.Client {
	clientOptions := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Cannot connect to MongoDB client %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Cannot ping MongoDB %v", err)
	}

	return client
}
