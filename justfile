# POSTFACTA

COVERAGE_OUTPUT := "coverage.out"

# List out available commands
_default:
	@just --list --unsorted

# Run the reposcout agent locally
start:
    #!/usr/bin/env bash
    echo "[INFO] Starting reposcout agent..."
    go run .

# Run unit tests
test:
    @go clean -testcache
    go test -cover -race -shuffle=on ./...
