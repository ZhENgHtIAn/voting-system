package pkg

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func NewRedisClient(cfg RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

func PingRedis(ctx context.Context, client *redis.Client) error {
	if _, err := client.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("ping redis failed: %w", err)
	}
	return nil
}

func EnsureTopicsInitialized(ctx context.Context, client *redis.Client, hashKey string, topics []string) error {
	for _, topic := range topics {
		if _, err := client.HSetNX(ctx, hashKey, topic, 0).Result(); err != nil {
			return fmt.Errorf("initialize topic %q failed: %w", topic, err)
		}
	}
	return nil
}
