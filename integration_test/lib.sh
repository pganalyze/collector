# shellcheck shell=bash
#
# Helpers shared by the integration test scripts. Source this after setting
# TEST_NAME, which names the test's container and image.

set -eEuo pipefail
trap 'echo "$TEST_NAME: command failed (${BASH_SOURCE[0]}:$LINENO): $BASH_COMMAND" >&2' ERR

cd "$(dirname "${BASH_SOURCE[0]}")"

CONTAINER="pganalyze-collector-test-$TEST_NAME"

fail() {
  echo "$TEST_NAME: $*" >&2
  exit 1
}

cleanup() {
  docker rm -f "$CONTAINER" > /dev/null 2>&1 || true
  docker rmi "$CONTAINER" > /dev/null 2>&1 || true
}
trap cleanup EXIT

# Builds the test image with the repository root as the context. Arguments are
# passed to docker build.
build_image() {
  docker build -t "$CONTAINER" "$@" ..
}

# Starts the test image in the background. Arguments replace the image's
# command.
#
# The cgroup and privilege options let systemd run inside the container with
# cgroups v2 (check for /sys/fs/cgroup/cgroup.controllers on the host):
# - https://systemd.io/CONTAINER_INTERFACE/
# - https://serverfault.com/questions/1053187/systemd-fails-to-run-in-a-docker-container-when-using-cgroupv2-cgroupns-priva
start_container() {
  docker run --name "$CONTAINER" \
    --cgroup-parent=docker.slice --cgroupns private \
    --privileged --cap-add SYS_ADMIN --security-opt seccomp=unconfined \
    --tmpfs /tmp --tmpfs /run --tmpfs /run/lock \
    -v "$PWD/docker-entrypoint-initdb.d:/docker-entrypoint-initdb.d/" \
    -e POSTGRES_HOST_AUTH_METHOD=trust \
    -d "$CONTAINER" "$@"
}

wait_for_postgres() {
  # Check over TCP, since the image's entrypoint first runs a temporary server
  # that only listens on the Unix socket while it initializes the database.
  docker exec "$CONTAINER" timeout 60 bash -c \
    'until pg_isready --quiet --host 127.0.0.1 --username postgres; do sleep 0.5; done'
}

wait_for_systemd() {
  # "degraded" means some unit failed, which is common in containers and
  # doesn't matter for these tests.
  docker exec "$CONTAINER" timeout 60 bash -c \
    'until [[ $(systemctl is-system-running 2> /dev/null) =~ ^(running|degraded)$ ]]; do sleep 0.5; done'
}
