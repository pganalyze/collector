#!/bin/bash

set -e

# Remove all except the master signing key, so reprepro does the right thing (subkeys are not supported)
printf "key 1\ndelkey\ny\nkey 1\ndelkey\ny\nsave\n" | gpg --batch --command-fd 0 --edit-key $REPO_GPG_KEY

mkdir -p /repo/ubuntu/xenial/conf
cp /root/deb.distributions /repo/ubuntu/xenial/conf/distributions
reprepro --basedir /repo/ubuntu/xenial includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
reprepro --basedir /repo/ubuntu/xenial includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64

mkdir -p /repo/ubuntu/bionic/conf
cp /root/deb.distributions /repo/ubuntu/bionic/conf/distributions
reprepro --basedir /repo/ubuntu/bionic includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
reprepro --basedir /repo/ubuntu/bionic includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64

mkdir -p /repo/ubuntu/focal/conf
cp /root/deb.distributions /repo/ubuntu/focal/conf/distributions
reprepro --basedir /repo/ubuntu/focal includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
reprepro --basedir /repo/ubuntu/focal includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64

mkdir -p /repo/debian/jessie/conf
cp /root/deb.distributions /repo/debian/jessie/conf/distributions
reprepro --basedir /repo/debian/jessie includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
reprepro --basedir /repo/debian/jessie includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64

mkdir -p /repo/debian/stretch/conf
cp /root/deb.distributions /repo/debian/stretch/conf/distributions
reprepro --basedir /repo/debian/stretch includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
reprepro --basedir /repo/debian/stretch includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64

mkdir -p /repo/debian/buster/conf
cp /root/deb.distributions /repo/debian/buster/conf/distributions
reprepro --basedir /repo/debian/buster includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
reprepro --basedir /repo/debian/buster includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64

mkdir -p /repo/debian/bullseye/conf
cp /root/deb.distributions /repo/debian/bullseye/conf/distributions
reprepro --basedir /repo/debian/bullseye includedeb stable /deb/systemd/$DEB_PACKAGE_X86_64
reprepro --basedir /repo/debian/bullseye includedeb stable /deb/systemd/$DEB_PACKAGE_ARM64

# Verify signatures
apt-key add /repo/pganalyze_signing_key.asc
gpgv --keyring /etc/apt/trusted.gpg /repo/ubuntu/xenial/dists/stable/InRelease
gpgv --keyring /etc/apt/trusted.gpg /repo/ubuntu/bionic/dists/stable/InRelease
gpgv --keyring /etc/apt/trusted.gpg /repo/ubuntu/focal/dists/stable/InRelease
gpgv --keyring /etc/apt/trusted.gpg /repo/debian/jessie/dists/stable/InRelease
gpgv --keyring /etc/apt/trusted.gpg /repo/debian/stretch/dists/stable/InRelease
gpgv --keyring /etc/apt/trusted.gpg /repo/debian/buster/dists/stable/InRelease
gpgv --keyring /etc/apt/trusted.gpg /repo/debian/bullseye/dists/stable/InRelease
