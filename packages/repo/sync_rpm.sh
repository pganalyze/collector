#!/bin/bash

set -e

# Remove all subkeys, so gpg signs with the master key rather than the newer signing subkey
# it would pick by default. Subkeys were previously not handled correctly by RPM, and this
# keeps the key ID consistent with all previously published packages and repository metadata,
# and matches what sync_deb.sh does.
printf "key 1\ndelkey\ny\nkey 1\ndelkey\ny\nsave\n" | gpg --batch --command-fd 0 --edit-key $REPO_GPG_KEY

# We run without a TTY (via docker exec) and the key has no passphrase after export.
# Set GPG_TTY so rpmsign does not warn that it cannot determine the TTY from stdin.
export GPG_TTY=/dev/null

rpm --addsign /rpm/systemd/$RPM_PACKAGE_X86_64
rpm --addsign /rpm/systemd/$RPM_PACKAGE_ARM64

# Verify that we've actually correctly signed the packages
rpm --import https://packages.pganalyze.com/pganalyze_signing_key.asc
rpm --checksig -v /rpm/systemd/$RPM_PACKAGE_X86_64
rpm --checksig -v /rpm/systemd/$RPM_PACKAGE_ARM64

# The list of distributions is passed in from the Makefile
if [ -z "$RPM_DISTROS" ]; then
  echo "RPM_DISTROS is not set" >&2
  exit 1
fi

for distro in $RPM_DISTROS; do
  mkdir -p /repo/$distro/RPMS
  cp /rpm/systemd/$RPM_PACKAGE_X86_64 /repo/$distro/RPMS/
  cp /rpm/systemd/$RPM_PACKAGE_ARM64 /repo/$distro/RPMS/
  createrepo --update /repo/$distro
  rm -f /repo/$distro/repodata/repomd.xml.asc
  gpg --detach-sign --armor --batch /repo/$distro/repodata/repomd.xml
done
