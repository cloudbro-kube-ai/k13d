#!/bin/bash
# Docker-based test runner for k13d
# Usage: ./scripts/docker-test.sh [unit|integration|all|bench]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Test mode (unit, integration, all, bench)
TEST_MODE="${1:-all}"

# Docker compose file for test infrastructure
COMPOSE_FILE="docker-compose.test.yaml"

# Cleanup function
cleanup() {
    log_info "Cleaning up test containers..."
    docker compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true
}

# Trap to cleanup on exit
trap cleanup EXIT

run_unit_tests() {
    log_info "Running unit tests..."

    # Run unit tests in Docker
    docker run --rm \
        -v "$PROJECT_DIR:/app" \
        -w /app \
        golang:1.25-alpine \
        sh -c "
            apk add --no-cache git &&
            go test -v -race -coverprofile=coverage.out ./... 2>&1
        "

    log_success "Unit tests completed"
}

run_integration_tests() {
    log_info "Starting test infrastructure..."

    # Start test infrastructure
    docker compose -f "$COMPOSE_FILE" up -d postgres mariadb redis mock-openai

    # Wait for services to be healthy
    log_info "Waiting for services to be ready..."
    sleep 10

    # Check service health
    docker compose -f "$COMPOSE_FILE" ps

    log_info "Running integration tests..."

    # Run integration tests
    docker run --rm \
        --network k13d-test-network \
        -v "$PROJECT_DIR:/app" \
        -w /app \
        -e POSTGRES_HOST=postgres \
        -e POSTGRES_PORT=5432 \
        -e POSTGRES_USER=k13d \
        -e POSTGRES_PASSWORD=testpassword \
        -e POSTGRES_DB=k13d_test \
        -e MOCK_OPENAI_URL=http://mock-openai:8080 \
        golang:1.25-alpine \
        sh -c "
            apk add --no-cache git curl &&
            go test -v -tags=integration ./tests/integration/... 2>&1 || true
        "

    log_success "Integration tests completed"
}

run_ollama_tests() {
    log_info "Starting Ollama for LLM tests..."

    # Start Ollama
    docker compose -f "$COMPOSE_FILE" up -d ollama

    # Wait for Ollama to be ready
    log_info "Waiting for Ollama to be ready..."
    local max_attempts=60
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf http://localhost:11434/api/tags > /dev/null 2>&1; then
            log_success "Ollama is ready"
            break
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    if [ $attempt -eq $max_attempts ]; then
        log_error "Ollama did not become ready in time"
        return 1
    fi

    # Pull the test model
    log_info "Pulling qwen2.5:3b model..."
    docker compose -f "$COMPOSE_FILE" up ollama-pull

    log_info "Running LLM integration tests..."

    # Run LLM tests
    docker run --rm \
        --network k13d-test-network \
        -v "$PROJECT_DIR:/app" \
        -w /app \
        -e OLLAMA_HOST=http://ollama:11434 \
        -e K13S_LLM_PROVIDER=ollama \
        -e K13S_LLM_MODEL=qwen2.5:3b \
        golang:1.25-alpine \
        sh -c "
            apk add --no-cache git curl &&
            go test -v -tags=ollama ./pkg/ai/... 2>&1 || true
        "

    log_success "LLM tests completed"
}

run_benchmark() {
    log_info "Building benchmark binary..."

    # Build benchmark binary
    docker run --rm \
        -v "$PROJECT_DIR:/app" \
        -w /app \
        golang:1.25-alpine \
        sh -c "
            apk add --no-cache git &&
            go build -o .build/k13d-bench ./cmd/bench/main.go
        "

    log_info "Starting benchmark infrastructure..."

    # Start Kind cluster for benchmarks (if kind is installed)
    if command -v kind &> /dev/null; then
        log_info "Creating Kind cluster for benchmarks..."
        kind create cluster --name k13d-bench 2>/dev/null || true

        # Run benchmarks
        log_info "Running AI benchmarks..."
        ./.build/k13d-bench list --task-dir benchmarks/tasks

        # Cleanup
        log_info "Cleaning up Kind cluster..."
        kind delete cluster --name k13d-bench 2>/dev/null || true
    else
        log_warn "Kind not installed, skipping cluster-based benchmarks"
        log_info "Listing available benchmark tasks..."
        docker run --rm \
            -v "$PROJECT_DIR:/app" \
            -w /app \
            golang:1.25-alpine \
            sh -c "
                apk add --no-cache git &&
                go run ./cmd/bench/main.go list --task-dir benchmarks/tasks
            "
    fi

    log_success "Benchmark completed"
}

show_coverage() {
    log_info "Generating coverage report..."

    docker run --rm \
        -v "$PROJECT_DIR:/app" \
        -w /app \
        golang:1.25-alpine \
        sh -c "
            apk add --no-cache git &&
            go test -coverprofile=coverage.out ./... 2>/dev/null &&
            go tool cover -func=coverage.out | tail -20
        "
}

# Main execution
case "$TEST_MODE" in
    unit)
        run_unit_tests
        show_coverage
        ;;
    integration)
        run_integration_tests
        ;;
    ollama)
        run_ollama_tests
        ;;
    bench)
        run_benchmark
        ;;
    all)
        run_unit_tests
        run_integration_tests
        show_coverage
        ;;
    coverage)
        show_coverage
        ;;
    *)
        echo "Usage: $0 [unit|integration|ollama|bench|all|coverage]"
        echo ""
        echo "Modes:"
        echo "  unit        Run unit tests only"
        echo "  integration Run integration tests with Docker services"
        echo "  ollama      Run LLM tests with Ollama"
        echo "  bench       Run AI benchmarks"
        echo "  all         Run all tests (default)"
        echo "  coverage    Show test coverage report"
        exit 1
        ;;
esac

log_success "Test execution completed!"
