#!/usr/bin/env bash
#
# Runs the guided setup against a fresh Postgres install, then checks the
# Postgres settings and collector config it wrote, and that the collector works
# with them.

TEST_NAME=guided-setup
# shellcheck source=lib.sh
source "$(dirname "$0")/lib.sh"

build_image -f Dockerfile.test-systemd --target guided-setup
start_container
wait_for_systemd

docker exec "$CONTAINER" systemctl start postgresql
docker exec --user pganalyze --detach "$CONTAINER" pganalyze-collector

docker exec --privileged --env PGA_SETUP_COLLECTOR_TEST_EXTRA_ARGS=--dry-run "$CONTAINER" \
  pganalyze-collector-setup --api-key=abc123 --recommended --db-name=postgres

docker exec --user postgres "$CONTAINER" psql \
  -c "SHOW shared_preload_libraries" \
  -c "SELECT count(*) FROM pg_settings WHERE pending_restart" \
  -c "SELECT name, setting FROM pg_settings WHERE name IN ('log_duration', 'log_error_verbosity', 'log_line_prefix', 'log_min_duration_statement', 'log_statement', 'auto_explain.log_analyze', 'auto_explain.log_buffers', 'auto_explain.log_timing', 'auto_explain.log_triggers', 'auto_explain.log_verbose', 'auto_explain.log_format', 'auto_explain.log_min_duration', 'auto_explain.log_nested_statements') ORDER BY name" \
  -c "SELECT pg_has_role('pganalyze', 'pg_monitor', 'usage')" \
  > guided-setup.postgres.out
docker exec --user pganalyze "$CONTAINER" cat /etc/pganalyze-collector.conf > guided-setup.collector-config.out
docker exec --env PGA_DISABLE_ACTIVITY=1 "$CONTAINER" pganalyze-collector --dry-run --force-state-update -v \
  > guided-setup.snapshot.json.out 2> guided-setup-snapshot.log.out

docker exec --user postgres "$CONTAINER" pg_ctl --pgdata /var/lib/postgresql/14/main --wait stop --mode fast

diff -Nau guided-setup.postgres.expected guided-setup.postgres.out
echo 'postgres configuration modified as expected'

grep -v db_password guided-setup.collector-config.out > guided-setup.collector-config.nopass.out
diff -Nau guided-setup.collector-config.nopass.expected guided-setup.collector-config.nopass.out
if ! grep --extended-regexp --quiet 'db_password\s+=\s+[a-f0-9]{32}' guided-setup.collector-config.out; then
  fail "expected a generated db_password in the collector config"
fi
echo 'collector config file modified as expected'

if [[ $(grep -c 'Local log test successful' guided-setup-snapshot.log.out) -ne 2 ]]; then
  cat guided-setup-snapshot.log.out
  fail "expected post-setup log test to succeed with both full and reduced privileges"
fi
if grep --quiet --extended-regexp '[0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} W ' guided-setup-snapshot.log.out; then
  cat guided-setup-snapshot.log.out
  fail "expected no warnings in collector log"
fi

jq '{roles: .roleReferences[] | select(.name == "pganalyze")}' < guided-setup.snapshot.json.out > guided-setup.snapshot-subset.json.out
diff -Nau guided-setup.snapshot-subset.json.expected guided-setup.snapshot-subset.json.out

echo "$TEST_NAME: success"
