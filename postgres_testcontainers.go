//go:build integration

package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// startPostgresContainer spins up a PostgreSQL container for testing.
// It sets KVDB_POSTGRES_PASSWORD so Init() can read it via os.Getenv.
// Call PostgresTeardown(ctx) when done.
func startPostgresContainer(ctx context.Context, t *testing.T) (host string, port uint16, teardown func(ctx context.Context), err error) {
	testPassword := generateTestPassword()

	container, err := postgres.Run(ctx,
		"postgres:18.1-alpine",
		postgres.WithDatabase("kvdb-test"),
		postgres.WithUsername("kvdb"),
		postgres.WithPassword(testPassword),
	)
	if err != nil {
		return "", 0, nil, fmt.Errorf("postgres container: %w", err)
	}

	host, err = container.Host(ctx)
	if err != nil {
		return "", 0, nil, fmt.Errorf("postgres host: %w", err)
	}

	portRaw, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return "", 0, nil, fmt.Errorf("postgres port: %w", err)
	}
	port = uint16(portRaw.Int())

	os.Setenv("KVDB_POSTGRES_PASSWORD", testPassword)

	teardown = func(ctx2 context.Context) {
		_ = container.Terminate(ctx2)
	}

	return host, port, teardown, nil
}

func generateTestPassword() string {
	// Same length as Jenkinsfile's generateAplhaNumericString(16)
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 16)
	for i := range result {
		result[i] = letters[i%len(letters)]
	}
	return string(result)
}
