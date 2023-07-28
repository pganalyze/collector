#!/bin/bash

set -e

# Remove all except the master signing key, so reprepro does the right thing (subkeys are not supported)
printf "key 1\ndelkey\ny\nkey 1\ndelkey\ny\nsave\n" | gpg --batch --command-fd 0 --edit-key $REPO_GPG_KEY

mkdir -p /repo/ubuntu/focal/conf
cp /root/deb.distributions /repo/ubuntu/focal/conf/distributions
reprepro --basedir /repo/ubuntu/focal includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
reprepro --basedir /repo/ubuntu/focal includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64

mkdir -p /repo/ubuntu/jammy/conf
cp /root/deb.distributions /repo/ubuntu/jammy/conf/distributions
reprepro --basedir /repo/ubuntu/jammy includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
reprepro --basedir /repo/ubuntu/jammy includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64

mkdir -p /repo/debian/bullseye/conf
cp /root/deb.distributions /repo/debian/bullseye/conf/distributions
reprepro --basedir /repo/debian/bullseye includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
reprepro --basedir /repo/debian/bullseye includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64

mkdir -p /repo/debian/bookworm/conf
cp /root/deb.distributions /repo/debian/bookworm/conf/distributions
reprepro --basedir /repo/debian/bookworm includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
reprepro --basedir /repo/debian/bookworm includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64

# Verify signatures
apt-key add /repo/pganalyze_signing_key.asc
gpgv --keyring /etc/apt/trusted.gpg /repo/ubuntu/focal/dists/stable/InRelease
gpgv --keyring /etc/apt/trusted.gpg /repo/ubuntu/jammy/dists/stable/InRelease
gpgv --keyring /etc/apt/trusted.gpg /repo/debian/bullseye/dists/stable/InRelease
gpgv --keyring /etc/apt/trusted.gpg /repo/debian/bookworm/dists/stable/InRelease
