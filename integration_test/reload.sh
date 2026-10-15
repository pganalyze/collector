#!/usr/bin/env bash
#
# Tests reloading both with an empty config and with a server present.
#
# We reload twice after adding a server to the config, so we can test the
# "write state file when reload occurs" logic, which requires the server to be
# there *before* the reload. Since --reload only signals the running collector,
# which writes the state file asynchronously, we poll for it instead of
# checking once.

TEST_NAME=reload
# shellcheck source=lib.sh
source "$(dirname "$0")/lib.sh"

state_file=/var/lib/pganalyze-collector/state

build_image -f Dockerfile.test-systemd --target reload
start_container
wait_for_systemd

docker exec "$CONTAINER" systemctl start pganalyze-collector
docker exec "$CONTAINER" pganalyze-collector --reload
docker exec "$CONTAINER" journalctl -u pganalyze-collector | grep "Reloading configuration..."
docker exec "$CONTAINER" test ! -f "$state_file"

docker exec "$CONTAINER" bash -c "echo 'db_url = postgres://127.0.0.1/postgres' >> /etc/pganalyze-collector.conf"
docker exec "$CONTAINER" pganalyze-collector --reload
docker exec "$CONTAINER" pganalyze-collector --reload
docker exec "$CONTAINER" timeout 30 bash -c "until test -f $state_file; do sleep 0.1; done"

echo "$TEST_NAME: success"
