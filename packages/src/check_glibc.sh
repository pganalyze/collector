#!/bin/sh
#
# Usage: check_glibc.sh MAX_GLIBC BINARY...
#
# Fails if any of the given binaries references a glibc symbol version newer
# than MAX_GLIBC. This guards against accidentally raising the minimum glibc
# version required by our packages, e.g. through a newer build image or a
# cgo dependency pulling in newer symbols. See packages/README.md.

set -e

max=$1
shift

for binary in "$@"; do
  need=$(objdump -T "$binary" 2>/dev/null | grep -oE 'GLIBC_[0-9]+\.[0-9]+' | sed 's/^GLIBC_//' | sort -V | tail -1)
  if [ -z "$need" ]; then
    echo "$binary: statically linked, OK"
    continue
  fi
  if [ "$(printf '%s\n%s\n' "$need" "$max" | sort -V | tail -1)" != "$max" ]; then
    echo "ERROR: $binary requires glibc $need, but the maximum allowed is $max" >&2
    exit 1
  fi
  echo "$binary: requires glibc $need (maximum allowed $max), OK"
done
