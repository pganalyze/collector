#!/bin/bash

set -e

# Remove all except the master signing key, so reprepro does the right thing (subkeys are not supported)
printf "key 1\ndelkey\ny\nkey 1\ndelkey\ny\nsave\n" | gpg --batch --command-fd 0 --edit-key $REPO_GPG_KEY

# The list of distributions is passed in from the Makefile
if [ -z "$DEB_DISTROS" ]; then
  echo "DEB_DISTROS is not set" >&2
  exit 1
fi

for distro in $DEB_DISTROS; do
  mkdir -p /repo/$distro/conf
  cp /root/deb.distributions /repo/$distro/conf/distributions
  reprepro --basedir /repo/$distro includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
  reprepro --basedir /repo/$distro includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64
done

# Verify signatures
gpg --dearmor --yes -o /repo/pganalyze_signing_key.gpg /repo/pganalyze_signing_key.asc
for distro in $DEB_DISTROS; do
  gpgv --keyring /repo/pganalyze_signing_key.gpg /repo/$distro/dists/stable/InRelease
done
