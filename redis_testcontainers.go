//go:build integration

package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/redis"
)

// startRedisContainer spins up a Redis container for testing.
// It sets KVDB_REDIS_PASSWORD so Init() can read it via os.Getenv.
// Call RedisTeardown(ctx) when done.
func startRedisContainer(ctx context.Context, t *testing.T) (host string, port uint16, teardown func(ctx context.Context), err error) {
	testPassword := generateTestPassword()

	container, err := redis.Run(ctx,
		"redis:8.2.3",
		redis.WithPassword(testPassword),
	)
	if err != nil {
		return "", 0, nil, fmt.Errorf("redis container: %w", err)
	}

	host, err = container.Host(ctx)
	if err != nil {
		return "", 0, nil, fmt.Errorf("redis host: %w", err)
	}

	portRaw, err := container.MappedPort(ctx, "6379")
	if err != nil {
		return "", 0, nil, fmt.Errorf("redis port: %w", err)
	}
	port = uint16(portRaw.Int())

	os.Setenv("KVDB_REDIS_PASSWORD", testPassword)

	teardown = func(ctx2 context.Context) {
		_ = container.Terminate(ctx2)
	}

	return host, port, teardown, nil
}