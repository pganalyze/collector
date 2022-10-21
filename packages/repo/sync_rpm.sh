#!/bin/bash

set -e

# Remove all except the master signing key, so rpm does the right thing (subkeys are not supported)
printf "key 1\ndelkey\ny\nkey 1\ndelkey\ny\nsave\n" | gpg --batch --command-fd 0 --edit-key $REPO_GPG_KEY

rpm --addsign /rpm/systemd/$RPM_PACKAGE_X86_64
rpm --addsign /rpm/systemd/$RPM_PACKAGE_ARM64

# Verify that we've actually correctly signed the packages
rpm --import https://packages.pganalyze.com/pganalyze_signing_key.asc
rpm --checksig /rpm/systemd/$RPM_PACKAGE_X86_64
rpm --checksig /rpm/systemd/$RPM_PACKAGE_ARM64

mkdir -p /repo/el/7/RPMS
cp /rpm/systemd/$RPM_PACKAGE_X86_64 /repo/el/7/RPMS/
cp /rpm/systemd/$RPM_PACKAGE_ARM64 /repo/el/7/RPMS/
createrepo --update /repo/el/7
rm -f /repo/el/7/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/el/7/repodata/repomd.xml

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

mkdir -p /repo/fedora/34/RPMS
cp /rpm/systemd/$RPM_PACKAGE_X86_64 /repo/fedora/34/RPMS/
cp /rpm/systemd/$RPM_PACKAGE_ARM64 /repo/fedora/34/RPMS/
createrepo --update /repo/fedora/34
rm -f /repo/fedora/34/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/fedora/34/repodata/repomd.xml

mkdir -p /repo/fedora/35/RPMS
cp /rpm/systemd/$RPM_PACKAGE_X86_64 /repo/fedora/35/RPMS/
cp /rpm/systemd/$RPM_PACKAGE_ARM64 /repo/fedora/35/RPMS/
createrepo --update /repo/fedora/35
rm -f /repo/fedora/35/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/fedora/35/repodata/repomd.xml

mkdir -p /repo/fedora/36/RPMS
cp /rpm/systemd/$RPM_PACKAGE_X86_64 /repo/fedora/36/RPMS/
cp /rpm/systemd/$RPM_PACKAGE_ARM64 /repo/fedora/36/RPMS/
createrepo --update /repo/fedora/36
rm -f /repo/fedora/36/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/fedora/36/repodata/repomd.xml
