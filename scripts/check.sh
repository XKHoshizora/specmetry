#!/bin/sh
# Runs the same checks as CI on the local machine.
# Requires Go (version in go.mod), Node.js >= 22.12, git and golangci-lint v2.
set -eu
cd "$(dirname "$0")/.."

echo "== go mod tidy"
go mod tidy -diff
echo "== go vet"
go vet -tags nowebui ./...
echo "== golangci-lint"
golangci-lint run ./...
echo "== go test"
go test -tags nowebui -count=1 ./...

echo "== web"
(cd web && npm ci && npm run check)

echo "== build with dashboard"
go build -o dist/ ./cmd/specmetry
go test -count=1 ./web/
echo "all checks passed"
