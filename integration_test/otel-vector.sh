#!/usr/bin/env bash
#
# Sends test logs through Vector with the CloudNativePG remap from
# contrib/vector to the collector's OpenTelemetry endpoint.

TEST_NAME=otel-vector
# shellcheck source=lib.sh
source "$(dirname "$0")/lib.sh"

build_image -f Dockerfile.test-postgres --build-arg BASE_IMAGE=postgres:17 --build-arg INSTALL_VECTOR=true
start_container
wait_for_postgres

docker exec --detach "$CONTAINER" vector \
  --config /collector/integration_test/otel-vector/vector.yaml \
  --config /collector/contrib/vector/cnpg_remap.yaml

docker exec --env DB_USERNAME=postgres --env PGA_API_KEY=dummy --env LOG_OTEL_SERVER=127.0.0.1:4318 "$CONTAINER" \
  pganalyze-collector --dry-run-logs -v > otel-vector.out 2>&1 || true
if ! grep -q 'Log test successful' otel-vector.out; then
  cat otel-vector.out
  fail "expected log test to succeed"
fi

echo "$TEST_NAME: success"
