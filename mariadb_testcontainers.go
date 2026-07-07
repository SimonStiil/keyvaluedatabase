//go:build integration

package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

// startMariaDBContainer spins up a MariaDB container for testing.
// It sets KVDB_MYSQL_PASSWORD so Init() can read it via os.Getenv.
// Call MariaDBTeardown(ctx) when done.
func startMariaDBContainer(ctx context.Context, t *testing.T) (host string, port uint16, teardown func(ctx context.Context), err error) {
	testPassword := generateTestPassword()

	container, err := mysql.Run(ctx,
		"mariadb:11.3.2-jammy",
		mysql.WithDatabase("kvdb-test"),
		mysql.WithUsername("kvdb"),
		mysql.WithPassword(testPassword),
	)
	if err != nil {
		return "", 0, nil, fmt.Errorf("mariadb container: %w", err)
	}

	host, err = container.Host(ctx)
	if err != nil {
		return "", 0, nil, fmt.Errorf("mariadb host: %w", err)
	}

	portRaw, err := container.MappedPort(ctx, "3306")
	if err != nil {
		return "", 0, nil, fmt.Errorf("mariadb port: %w", err)
	}
	port = uint16(portRaw.Int())

	os.Setenv("KVDB_MYSQL_PASSWORD", testPassword)

	teardown = func(ctx2 context.Context) {
		_ = container.Terminate(ctx2)
	}

	return host, port, teardown, nil
}