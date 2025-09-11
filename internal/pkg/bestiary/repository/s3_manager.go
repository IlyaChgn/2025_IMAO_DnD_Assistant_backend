package repository

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
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

func (m *minioManager) UploadImage(ctx context.Context, base64Data string, objectName string) (string, error) {
	l := logger.FromContext(ctx)

	parts := strings.Split(base64Data, ",")
	if len(parts) != 2 {
		l.RepoWarn(apperrors.InvalidBase64Err, nil)
		return "", apperrors.InvalidBase64Err
	}

	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		l.RepoError(err, nil)
		return "", err
	}

	reader := bytes.NewReader(data)

	_, err = m.client.PutObject(ctx, m.bucketName, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "image/webp", // TODO: auto detect
	})
	if err != nil {
		l.RepoError(err, nil)
		return "", err
	}

	return fmt.Sprintf("https://%s/%s/%s", "encounterium.ru", m.bucketName, objectName), nil
}
