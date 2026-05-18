#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()   { echo -e "${GREEN}[$(date +'%H:%M:%S')]${NC} $*"; }
warn()  { echo -e "${YELLOW}[$(date +'%H:%M:%S')] WARN:${NC} $*"; }
error() { echo -e "${RED}[$(date +'%H:%M:%S')] ERROR:${NC} $*"; }

# Prerequisites
if ! command -v docker &>/dev/null; then
  error "docker is not installed or not in PATH"
  exit 1
fi
if ! command -v go &>/dev/null; then
  error "go is not installed or not in PATH"
  exit 1
fi
if ! command -v openssl &>/dev/null; then
  error "openssl is not installed or not in PATH"
  exit 1
fi

TESTPASSWORD=$(openssl rand -hex 12)
ROOTPASSWORD=$(openssl rand -hex 12)

MARIADB_CONTAINER="kvdb-test-mariadb"
POSTGRES_CONTAINER="kvdb-test-postgres"
REDIS_CONTAINER="kvdb-test-redis"
AUTHELIA_CONTAINER="kvdb-test-authelia"

cleanup() {
  log "Removing test containers..."
  docker rm -f "$MARIADB_CONTAINER" "$POSTGRES_CONTAINER" "$REDIS_CONTAINER" "$AUTHELIA_CONTAINER" 2>/dev/null || true
}
trap cleanup EXIT

# Remove any leftover containers from a previous run
docker rm -f "$MARIADB_CONTAINER" "$POSTGRES_CONTAINER" "$REDIS_CONTAINER" "$AUTHELIA_CONTAINER" 2>/dev/null || true

log "Starting MariaDB..."
docker run -d --name "$MARIADB_CONTAINER" \
  -p 127.0.0.1:3306:3306 \
  -e MARIADB_USER=kvdb \
  -e "MARIADB_PASSWORD=$TESTPASSWORD" \
  -e MARIADB_DATABASE=kvdb-test \
  -e "MARIADB_ROOT_PASSWORD=$ROOTPASSWORD" \
  mariadb:11.3.2-jammy

log "Starting PostgreSQL..."
docker run -d --name "$POSTGRES_CONTAINER" \
  -p 127.0.0.1:5432:5432 \
  -e POSTGRES_USER=kvdb \
  -e "POSTGRES_PASSWORD=$TESTPASSWORD" \
  -e POSTGRES_DB=kvdb-test \
  postgres:18.1-alpine

log "Starting Redis..."
docker run -d --name "$REDIS_CONTAINER" \
  -p 127.0.0.1:6379:6379 \
  redis:8.2.3 \
  redis-server --requirepass "$TESTPASSWORD"

log "Starting Authelia..."
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
docker run -d --name "$AUTHELIA_CONTAINER" \
  -p 127.0.0.1:19091:9091 \
  -v "$SCRIPT_DIR/testdata/authelia/configuration.yml:/config/configuration.yml:ro" \
  -v "$SCRIPT_DIR/testdata/authelia/users_database.yml:/config/users_database.yml:ro" \
  authelia/authelia:4.38

wait_for() {
  local name=$1
  local cmd=$2
  local max_attempts=30
  log "Waiting for $name to be ready..."
  for i in $(seq 1 $max_attempts); do
    if eval "$cmd" &>/dev/null; then
      log "$name is ready"
      return 0
    fi
    sleep 2
  done
  error "$name did not become ready after $((max_attempts * 2)) seconds"
  return 1
}

wait_for "MariaDB"    "docker exec $MARIADB_CONTAINER mariadb -u kvdb -p$TESTPASSWORD kvdb-test -e 'SELECT 1'"
wait_for "PostgreSQL" "docker exec $POSTGRES_CONTAINER pg_isready -U kvdb"
wait_for "Redis"      "docker exec $REDIS_CONTAINER redis-cli -a $TESTPASSWORD ping"
wait_for "Authelia"   "curl -sf http://127.0.0.1:19091/.well-known/openid-configuration"

log "Generating test TLS certificates..."
./generate-test-cert.sh

log "Running all tests (including container-backed and OIDC)..."
export CGO_ENABLED=0
export KVDB_DATABASETYPE=postgres
export KVDB_MYSQL_PASSWORD="$TESTPASSWORD"
export KVDB_POSTGRES_PASSWORD="$TESTPASSWORD"
export KVDB_REDIS_PASSWORD="$TESTPASSWORD"
export TEST_AUTHelia=true

go test . -v -tags="unit integration" -covermode=atomic -coverprofile=coverage.out
go tool cover -func coverage.out
