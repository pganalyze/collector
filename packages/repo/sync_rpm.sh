#!/bin/bash

set -e

# Remove all except the master signing key, so rpm does the right thing (subkeys are not supported)
printf "key 1\ndelkey\ny\nkey 1\ndelkey\ny\nsave\n" | gpg --batch --command-fd 0 --edit-key $REPO_GPG_KEY

rpm --addsign /rpm/sysvinit/$RPM_PACKAGE
rpm --addsign /rpm/systemd/$RPM_PACKAGE

# Verify that we've actually correctly signed the packages
rpm --import https://keybase.io/pganalyze/key.asc
rpm --checksig /rpm/sysvinit/$RPM_PACKAGE
rpm --checksig /rpm/systemd/$RPM_PACKAGE

mkdir -p /repo/el/6/RPMS
cp /rpm/sysvinit/$RPM_PACKAGE /repo/el/6/RPMS/
createrepo --update /repo/el/6
gpg --detach-sign --armor /repo/el/6/repodata/repomd.xml

mkdir -p /repo/el/7/RPMS
cp /rpm/systemd/$RPM_PACKAGE /repo/el/7/RPMS/
createrepo --update /repo/el/7
gpg --detach-sign --armor /repo/el/7/repodata/repomd.xml

mkdir -p /repo/fedora/24/RPMS
cp /rpm/systemd/$RPM_PACKAGE /repo/fedora/24/RPMS/
createrepo --update /repo/fedora/24
gpg --detach-sign --armor /repo/fedora/24/repodata/repomd.xml
