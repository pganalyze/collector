#!/usr/bin/env bash
#
# Runs contrib/install.sh, which installs the latest published package.

TEST_NAME=installer
# shellcheck source=lib.sh
source "$(dirname "$0")/lib.sh"

build_image -f Dockerfile.test-systemd --target guided-setup
start_container
wait_for_systemd

docker exec "$CONTAINER" systemctl start postgresql

if ! docker exec --privileged --env PGA_INSTALL_NONINTERACTIVE=true --env PGA_API_KEY=abc123 "$CONTAINER" \
  bash -c '</collector/contrib/install.sh bash' > install.out \
  || ! grep -q 'The pganalyze collector was installed successfully' install.out; then
  cat install.out
  fail "expected installer to install latest published package"
fi

echo "$TEST_NAME: success"
