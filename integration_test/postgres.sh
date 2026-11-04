#!/usr/bin/env bash
#
# Runs pgbench, takes two collector snapshots, and checks the second one for
# collector errors, for the table excluded by IGNORE_SCHEMA_REGEXP, and against
# the expected normalized queries in <test>.expected.
#
# Usage: ./postgres.sh pg12|pg13|pg14|pg15|pg16|pg17|citus

TEST_NAME=$1
# shellcheck source=lib.sh
source "$(dirname "$0")/lib.sh"

snapshot_env=()

pgbench() {
  docker exec "$CONTAINER" pgbench -U postgres postgres "$@"
}

snapshot() {
  docker exec \
    --env DB_USERNAME=postgres --env PGA_API_KEY=dummy --env PGA_DISABLE_ACTIVITY=1 \
    --env IGNORE_SCHEMA_REGEXP=pgbench_history \
    "${snapshot_env[@]}" \
    "$CONTAINER" pganalyze-collector --dry-run --force-state-update -v
}

case $TEST_NAME in
  citus) base_image=citusdata/citus:12.1 ;;
  pg*) base_image=postgres:${TEST_NAME#pg} ;;
  *) fail "unknown test" ;;
esac

build_image -f Dockerfile.test-postgres --build-arg BASE_IMAGE="$base_image"
start_container postgres -c pg_stat_statements.track_utility=off
wait_for_postgres

pgbench -i
if [[ $TEST_NAME == citus ]]; then
  docker exec -i "$CONTAINER" psql -U postgres postgres < citus-pgbench-ddl.sql
  # Citus is installed per database, so schema collection must also work for
  # databases on the same server that don't have it
  docker exec "$CONTAINER" psql -U postgres -c "CREATE DATABASE non_citus"
  docker exec "$CONTAINER" psql -U postgres non_citus -c "CREATE TABLE non_citus_table (id int PRIMARY KEY)"
  snapshot_env+=(--env DB_ALL_NAMES=1)
fi
pgbench
snapshot > "${TEST_NAME}_1.json"
pgbench
snapshot > "${TEST_NAME}_2.json"

result="${TEST_NAME}_2.json"

errors=$(jq '.collectorErrors' "$result")
if [[ $errors != null ]]; then
  fail "unexpected collector errors: $errors"
fi

relations=$(jq '.relationReferences' "$result")
if [[ $relations == *pgbench_history* ]]; then
  fail "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history: $relations"
fi

if [[ $TEST_NAME == citus ]]; then
  {
    # Shard names end in a generated shard ID
    jq '.queryInformations[].normalizedQuery' "$result" \
      | sed -E 's/pgbench_branches_[0-9]+/pgbench_branches_XXXXX/' \
      | sort --version-sort -f
    jq '.relationReferences[].relationName' "$result" | sort
  } > "$TEST_NAME.out"
else
  jq '.queryInformations[].normalizedQuery' "$result" | sort --version-sort -f > "$TEST_NAME.out"
fi
diff -Naur "$TEST_NAME.expected" "$TEST_NAME.out"

echo "$TEST_NAME: success"
