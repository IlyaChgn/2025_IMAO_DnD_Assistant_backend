package repository

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(host, port, password string, db int) *redis.Client {
	return redis.NewClient(
		&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", host, port),
			Password: password,
			DB:       db,
		},
	)
}
