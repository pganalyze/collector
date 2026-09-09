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

mkdir -p /repo/el/8/RPMS
cp /rpm/systemd/$RPM_PACKAGE_X86_64 /repo/el/8/RPMS/
cp /rpm/systemd/$RPM_PACKAGE_ARM64 /repo/el/8/RPMS/
createrepo --update /repo/el/8
rm -f /repo/el/8/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/el/8/repodata/repomd.xml

mkdir -p /repo/el/9/RPMS
cp /rpm/systemd/$RPM_PACKAGE_X86_64 /repo/el/9/RPMS/
cp /rpm/systemd/$RPM_PACKAGE_ARM64 /repo/el/9/RPMS/
createrepo --update /repo/el/9
rm -f /repo/el/9/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/el/9/repodata/repomd.xml

mkdir -p /repo/el/10/RPMS
cp /rpm/systemd/$RPM_PACKAGE_X86_64 /repo/el/10/RPMS/
cp /rpm/systemd/$RPM_PACKAGE_ARM64 /repo/el/10/RPMS/
createrepo --update /repo/el/10
rm -f /repo/el/10/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/el/10/repodata/repomd.xml

mkdir -p /repo/fedora/42/RPMS
cp /rpm/systemd/$RPM_PACKAGE_X86_64 /repo/fedora/42/RPMS/
cp /rpm/systemd/$RPM_PACKAGE_ARM64 /repo/fedora/42/RPMS/
createrepo --update /repo/fedora/42
rm -f /repo/fedora/42/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/fedora/42/repodata/repomd.xml

mkdir -p /repo/fedora/43/RPMS
cp /rpm/systemd/$RPM_PACKAGE_X86_64 /repo/fedora/43/RPMS/
cp /rpm/systemd/$RPM_PACKAGE_ARM64 /repo/fedora/43/RPMS/
createrepo --update /repo/fedora/43
rm -f /repo/fedora/43/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/fedora/43/repodata/repomd.xml
