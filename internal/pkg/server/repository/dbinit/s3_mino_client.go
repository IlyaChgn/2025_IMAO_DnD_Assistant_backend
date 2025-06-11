package dbinit

import (
	"context"
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinioEndpoint(host string, port ...string) string {
	if len(port) > 0 {
		return fmt.Sprintf("%s:%s", host, port[0])
	}
	return host
}

func ConnectToMinio(ctx context.Context, endpoint, accessKey, secretKey string, useSSL bool) *minio.Client {
	client := newMinioClient(ctx, endpoint, accessKey, secretKey, useSSL)
	return client
}

func newMinioClient(ctx context.Context, endpoint, accessKey, secretKey string, useSSL bool) *minio.Client {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %v", err)
	}

	_, err = client.ListBuckets(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to MinIO server: %v", err)
	}

	log.Println("Connected to MinIO at", endpoint)
	return client
}
