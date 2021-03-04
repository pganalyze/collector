#!/bin/bash

set -e

fail () {
  >&2 echo "Install failed: $1"
  >&2 echo
  >&2 echo "Please contact support@pganalyze.com for help and include information about your platform"
  exit 1
}

distribution=''
version=''
pkg=''

if ! test -r /etc/os-release;
then
  fail "cannot read /etc/os-release to determine distribution"
fi

arch=$(uname -m)
if [ "$arch" != 'x86_64' ];
then
  fail "unsupported architecture: $arch"
fi

if grep -q '^ID="amzn"$' /etc/os-release && grep -q '^VERSION_ID="2"$' /etc/os-release;
then
  # Amazon Linux 2, based on RHEL7
  pkg=yum
  distribution=el
  version=7
elif grep -q '^ID="rhel"$' /etc/os-release;
then
  # RHEL
  pkg=yum
  distribution=el
  version=$(grep VERSION_ID /etc/os-release | cut -d= -f2 | tr -d '"' | cut -d. -f1)
  if [ "$version" != 7 ] && [ "$version" != 8 ];
  then
    fail "unrecognized RHEL version: ${version}"
  fi
elif grep -q '^ID=fedora$' /etc/os-release;
then
  # Fedora
  pkg=yum
  distribution=fedora
  version=$(grep VERSION_ID /etc/os-release | cut -d= -f2)

  if [ "$version" != 30 ] && [ "$version" != 29 ];
  then
    fail "unrecognized Fedora version: ${version}"
  fi
elif grep -q '^ID=ubuntu$' /etc/os-release;
then
  # Ubuntu
  pkg=deb
  distribution=ubuntu
  version=$(grep VERSION_CODENAME /etc/os-release | cut -d= -f2)
  if [ "$version" != focal ] && [ "$version" != bionic ] && [ "$version" != xenial ];
  then
    fail "unrecognized Ubuntu version: ${version}"
  fi
elif grep -q '^ID=debian$' /etc/os-release;
then
  # Debian
  pkg=deb
  distribution=debian
  version=$(grep VERSION_CODENAME /etc/os-release | cut -d= -f2)
  if [ "$version" != buster ] && [ "$version" != stretch ];
  then
    fail "unrecognized Debian version: ${version}"
  fi
else
  >&2 cat /etc/os-release
  fail "unrecognized distribution: ${distribution}"
fi

# If we're already running as sudo or root, no need to do anything;
# if we're not, set up sudo for relevant commands
maybe_sudo=''
if [ "$(id -u)" != "0" ]; then
  maybe_sudo=$(command -v sudo)
  echo "This script requires superuser access to install packages"

  if [ -z "$maybe_sudo" ];
  then
    fail "not running as root and could not find sudo command"
  fi

  echo "You may be prompted for your password by sudo"

  # clear any previous sudo permission to avoid inadvertent confirmation
  $maybe_sudo -k
fi

if [ "$pkg" = yum ];
then
  echo "[pganalyze_collector]
name=pganalyze_collector
baseurl=https://packages.pganalyze.com/${distribution}/${version}
repo_gpgcheck=1
enabled=1
gpgkey=https://packages.pganalyze.com/pganalyze_signing_key.asc
sslverify=1
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
metadata_expire=300" | $maybe_sudo tee -a /etc/yum.repos.d/pganalyze_collector.repo
  $maybe_sudo yum makecache < /dev/tty
  $maybe_sudo yum install pganalyze-collector < /dev/tty
elif [ "$pkg" = deb ];
then
  apt_source="deb [arch=amd64] https://packages.pganalyze.com/${distribution}/${version}/ stable main"
  curl -L https://packages.pganalyze.com/pganalyze_signing_key.asc | $maybe_sudo apt-key add -
  echo "$apt_source" | $maybe_sudo tee /etc/apt/sources.list.d/pganalyze_collector.list
  $maybe_sudo apt-get update < /dev/tty
  $maybe_sudo apt-get install pganalyze-collector < /dev/tty
else
  fail "unrecognized package kind: $pkg"
fi

if [ -n "$PGA_API_KEY" ];
then
  $maybe_sudo sed -i "s/^#api_key = your_api_key$/api_key = ${PGA_API_KEY}/" /etc/pganalyze-collector.conf
fi

# run to validate install
pganalyze-collector --version

echo "The pganalyze collector has been installed"
