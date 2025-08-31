package dbinit

import (
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinioEndpoint(host string, port ...string) string {
	if len(port) > 0 {
		return fmt.Sprintf("%s:%s", host, port[0])
	}
	return host
}

func ConnectToMinio(endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	return newMinioClient(endpoint, accessKey, secretKey, useSSL)
}

func newMinioClient(endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}
