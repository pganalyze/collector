#!/bin/bash

set -e

# Remove all except the master signing key, so reprepro does the right thing (subkeys are not supported)
printf "key 1\ndelkey\ny\nkey 1\ndelkey\ny\nsave\n" | gpg --batch --command-fd 0 --edit-key $REPO_GPG_KEY

mkdir -p /repo/ubuntu/trusty/conf
cp /root/deb.distributions /repo/ubuntu/trusty/conf/distributions
reprepro --basedir /repo/ubuntu/trusty includedeb stable /deb/upstart/$DEB_PACKAGE

mkdir -p /repo/ubuntu/xenial/conf
cp /root/deb.distributions /repo/ubuntu/xenial/conf/distributions
reprepro --basedir /repo/ubuntu/xenial includedeb stable /deb/systemd/$DEB_PACKAGE

mkdir -p /repo/debian/jessie/conf
cp /root/deb.distributions /repo/debian/jessie/conf/distributions
reprepro --basedir /repo/debian/jessie includedeb stable /deb/systemd/$DEB_PACKAGE

mkdir -p /repo/debian/stretch/conf
cp /root/deb.distributions /repo/debian/stretch/conf/distributions
reprepro --basedir /repo/debian/stretch includedeb stable /deb/systemd/$DEB_PACKAGE
