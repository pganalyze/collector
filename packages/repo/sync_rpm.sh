#!/bin/bash

set -e

# Remove all except the master signing key, so rpm does the right thing (subkeys are not supported)
printf "key 1\ndelkey\ny\nkey 1\ndelkey\ny\nsave\n" | gpg --batch --command-fd 0 --edit-key $REPO_GPG_KEY

rpm --addsign /rpm/systemd/$RPM_PACKAGE

# Verify that we've actually correctly signed the packages
rpm --import https://keybase.io/pganalyze/key.asc
rpm --checksig /rpm/systemd/$RPM_PACKAGE

mkdir -p /repo/el/7/RPMS
cp /rpm/systemd/$RPM_PACKAGE /repo/el/7/RPMS/
createrepo --update /repo/el/7
rm -f /repo/el/7/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/el/7/repodata/repomd.xml

mkdir -p /repo/el/8/RPMS
cp /rpm/systemd/$RPM_PACKAGE /repo/el/8/RPMS/
createrepo --update /repo/el/8
rm -f /repo/el/8/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/el/8/repodata/repomd.xml

mkdir -p /repo/fedora/29/RPMS
cp /rpm/systemd/$RPM_PACKAGE /repo/fedora/29/RPMS/
createrepo --update /repo/fedora/29
rm -f /repo/fedora/29/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/fedora/29/repodata/repomd.xml

mkdir -p /repo/fedora/30/RPMS
cp /rpm/systemd/$RPM_PACKAGE /repo/fedora/30/RPMS/
createrepo --update /repo/fedora/30
rm -f /repo/fedora/30/repodata/repomd.xml.asc
gpg --detach-sign --armor --batch /repo/fedora/30/repodata/repomd.xml
