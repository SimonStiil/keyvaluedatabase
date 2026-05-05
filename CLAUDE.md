# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with this repository.

## Project Overview

KVDB is a web-based key-value store that serves as a webhook-compatible API. Originally built as a secrets storage backend for External Secrets in Kubernetes. It supports Basic auth with host-based access control, mTLS, and multiple pluggable database backends (YAML file, Redis, MySQL/MariaDB, PostgreSQL).

## Build and Run

```bash
go build .                          # Build binary
go run . -config=example-config     # Run with example config
go run . -generate=mypassword       # Generate a hashed password for config
```

## Testing

```bash
go test ./...                      # Run all tests
go test -run TestApiV1 -v          # Run a single test function
go test -run Test_Yaml_DB -v       # Run YAML backend tests (no external deps)
```

Tests for Redis, MySQL, and PostgreSQL backends require those services running. The YAML backend tests are self-contained and the safest to run locally without Docker. All tests use `setupTestlogging()` and the global `App` variable to bootstrap state — follow this pattern when adding new tests.

## Architecture

Everything is in `package main` at the root level. The key abstractions:

- **`Database` interface** (`database.go`) — pluggable backend with `Init`, `Set`, `Get`, `Keys`, `CreateNamespace`, `DeleteKey`, `DeleteNamespace`, `Close`. Four implementations: `YamlDatabase`, `RedisDatabase`, `MariaDatabase`, `PostgresDatabase`.
- **`API` interface** (`apinterface.go`) — versioned API endpoint. Two implementations: `APIv1` (`apiv1.go`) for the main KV operations and `Systemv1` (`systemv1.go`) for health/metrics.
- **`Application` struct** (`webapp.go`) — wires the `Auth`, `Database`, `Counter`, and `APIEndpoints` together. `RootControllerV1` routes every request through the API registry, runs authentication, then delegates to the matching API handler.
- **`Auth` struct** (`auth.go`) — Basic auth with per-user host allowlists and per-namespace read/write/list permissions. Also supports mTLS with client certificate verification and public-readable namespaces.
- **`RequestParameters`** (`requestparameters.go`) — enriched request wrapper with parsed namespace/key, decoded body, and per-request structured logging.
- **`rest/` package** — shared types (`ObjectV1`, `KVPairV2`, etc.) used for JSON serialization across API and handler layers.
- **`Counter`** (`counter.go`) — tracks request count, persisted to the configured database.

Configuration is via Viper with YAML file + environment variable override (prefix `KVDB_`). Config hot-reload is enabled via `fsnotify`.

## Key Conventions

- Custom HTTP methods `UPDATE` and `PATCH` are supported alongside standard methods, parsed in `httpauth.go` from the `X-HTTP-Method-Override` header.
- API paths follow `/v1/{namespace}/{key}` — namespace and key are parsed from the URL path in `GetRequestParameters`.
- Prompt caching is not applicable here — this is a Go binary, not an LLM application.
