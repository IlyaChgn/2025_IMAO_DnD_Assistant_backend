package repository

import (
	"bytes"
	"context"
	"fmt"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/minio/minio-go/v7"
)

// AvatarS3Manager handles avatar upload/delete operations in Minio S3.
type AvatarS3Manager interface {
	UploadAvatar(ctx context.Context, data []byte, objectName string) (string, error)
	DeleteAvatar(ctx context.Context, objectName string) error
}

type avatarS3Manager struct {
	client     *minio.Client
	bucketName string
}

func NewAvatarS3Manager(client *minio.Client, bucket string) AvatarS3Manager {
	return &avatarS3Manager{
		client:     client,
		bucketName: bucket,
	}
}

func (m *avatarS3Manager) UploadAvatar(ctx context.Context, data []byte, objectName string) (string, error) {
	l := logger.FromContext(ctx)

	reader := bytes.NewReader(data)

	_, err := m.client.PutObject(ctx, m.bucketName, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "image/webp",
	})
	if err != nil {
		l.RepoError(err, nil)
		return "", err
	}

	return fmt.Sprintf("https://%s/%s/%s", "encounterium.ru", m.bucketName, objectName), nil
}

func (m *avatarS3Manager) DeleteAvatar(ctx context.Context, objectName string) error {
	l := logger.FromContext(ctx)

	err := m.client.RemoveObject(ctx, m.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		l.RepoError(err, nil)
		return err
	}

	return nil
}
