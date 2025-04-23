package repository

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/minio/minio-go/v7"
)

type minioManager struct {
	client     *minio.Client
	bucketName string
}

func NewMinioManager(client *minio.Client, bucket string) bestiary.BestiaryS3Manager {
	return &minioManager{
		client:     client,
		bucketName: bucket,
	}
}

func (m *minioManager) UploadImage(base64Data string, objectName string) (string, error) {
	parts := strings.Split(base64Data, ",")
	if len(parts) != 2 {
		return "", errors.New("invalid base64 string")
	}

	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	reader := bytes.NewReader(data)
	_, err = m.client.PutObject(context.Background(), m.bucketName, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "image/webp", // TODO: auto detect
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s/%s/%s", "encounterium.ru", m.bucketName, objectName), nil
}
