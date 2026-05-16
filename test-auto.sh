#!/usr/bin/env sh
set -eu

run_integration=1
for arg in "$@"; do
	if [ "$arg" = "--skip-integration" ]; then
		run_integration=0
	fi
done

echo "[1/4] Running unit tests..."
go test ./...

echo "[2/4] Running vet..."
go vet ./...

if [ "$run_integration" = "0" ]; then
	echo "[3/4] Skipping integration tests because --skip-integration was provided."
	echo "[4/4] All checks passed."
	exit 0
fi

if [ -f .local.json ]; then
	echo "[3/4] Running integration tests..."
	go test -tags=integration -timeout 20m ./...
else
	echo "[3/4] Skipping integration tests because .local.json was not found."
fi

echo "[4/4] All checks passed."