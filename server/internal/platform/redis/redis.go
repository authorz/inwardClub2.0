// Package redis provides shared Redis connection options. asynq builds its own
// clients/servers from RedisOpt; other callers can use Client for generic use.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	goredis "github.com/redis/go-redis/v9"
)

// AsynqOpt returns the asynq Redis connection options for the given address.
func AsynqOpt(addr string) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{Addr: addr}
}

// NewClient returns a generic go-redis client and verifies connectivity.
func NewClient(ctx context.Context, addr string) (*goredis.Client, error) {
	client := goredis.NewClient(&goredis.Options{Addr: addr})
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}
