#!/bin/bash

set -e

fail () {
  >&2 echo
  >&2 echo "Install failed: $1"
  >&2 echo
  >&2 echo "Please contact support@pganalyze.com for help and include information about your platform"
  exit 1
}

user_input=''
yum_opts=''
apt_opts=''
pgags_opts=''
if [ -n "$PGA_INSTALL_NONINTERACTIVE" ];
then
  user_input=/dev/null
  apt_opts='--yes'
  yum_opts='--assumeyes'
  pgags_opts="--recommended --db-name=${DB_NAME:-postgres}"
else
  user_input=/dev/tty
fi

confirm () {
  if [ -n "$PGA_INSTALL_NONINTERACTIVE" ];
  then
    return 0
  fi

  local confirmation
  # N.B.: default is always yes
  read -r -n1 -p "$1 [Y/n]" confirmation <$user_input
  [ -z "$confirmation" ] || [[ "$confirmation" =~ [Yy] ]]
}

pkg=''
distribution=''
version=''

if ! test -r /etc/os-release;
then
  fail "cannot read /etc/os-release to determine distribution"
fi

arch=$(uname -m)
if [ "$arch" != 'x86_64' ] && [ "$arch" != 'arm64' ] && [ "$arch" != 'aarch64' ];
then
  fail "unsupported architecture: $arch"
fi

if grep -q '^ID="amzn"$' /etc/os-release && grep -q '^VERSION_ID="2"$' /etc/os-release;
then
  # Amazon Linux 2, based on RHEL7
  pkg=yum
  distribution=el
  version=7
elif grep -q '^ID="\(rhel\|centos\)"$' /etc/os-release;
then
  # RHEL and CentOS
  pkg=yum
  distribution=el
  version=$(grep VERSION_ID /etc/os-release | cut -d= -f2 | tr -d '"' | cut -d. -f1)
  if [ "$version" != 7 ] && [ "$version" != 8 ];
  then
    if confirm "Unsupported RHEL or CentOS version; try RHEL8 package?";
    then
      version=8
    else
      fail "unrecognized RHEL or CentOS version: ${version}"
    fi
  fi
elif grep -q '^ID=fedora$' /etc/os-release;
then
  # Fedora
  pkg=yum
  distribution=fedora
  version=$(grep VERSION_ID /etc/os-release | cut -d= -f2)

  if [ "$version" != 35 ] && [ "$version" != 34 ];
  then
    if confirm "Unsupported Fedora version; try Fedora 35 package?";
    then
      version=35
    else
      fail "unrecognized Fedora version: ${version}"
    fi
  fi
elif grep -q '^ID=ubuntu$' /etc/os-release;
then
  # Ubuntu
  pkg=deb
  distribution=ubuntu
  version=$(grep VERSION_CODENAME /etc/os-release | cut -d= -f2)
  if [ "$version" != jammy ] && [ "$version" != focal ] && [ "$version" != bionic ] && [ "$version" != xenial ];
  then
    if confirm "Unsupported Ubuntu version; try Ubuntu Focal (20.04) package?";
    then
      version=jammy
    else
      fail "unrecognized Ubuntu version: ${version}"
    fi
  fi
elif grep -q '^ID=debian$' /etc/os-release;
then
  # Debian
  pkg=deb
  distribution=debian
  version=$(grep VERSION_CODENAME /etc/os-release | cut -d= -f2)
  if [ "$version" != bullseye ] && [ "$version" != buster ] && [ "$version" != stretch ];
  then
    if confirm "Unsupported Debian version; try Debian Bullseye (11) package?";
    then
      version=bullseye
    else
      fail "unrecognized Debian version: ${version}"
    fi
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
  $maybe_sudo yum $yum_opts makecache <$user_input
  $maybe_sudo yum $yum_opts install pganalyze-collector <$user_input
elif [ "$pkg" = deb ];
then
  # on Debian, gnupg, required for apt-key add, is not installed by default, so install
  # it before trying to invoke it if necessary
  if ! dpkg --verify gnupg 2>/dev/null && ! dpkg --verify gnupg1 2>/dev/null && ! dpkg --verify gnupg2 2>/dev/null;
  then
    if confirm "The gnupg package is required to verify the collector package signature; install it now?";
    then
      $maybe_sudo apt-get $apt_opts update <$user_input
      $maybe_sudo apt-get $apt_opts install gnupg <$user_input
    else
      fail "cannot install without gnupg"
    fi
  fi
  if [ "$arch" = 'x86_64' ];
  then
    apt_source="deb [arch=amd64] https://packages.pganalyze.com/${distribution}/${version}/ stable main"
  elif [ "$arch" = 'arm64' ];
  then
    apt_source="deb [arch=arm64] https://packages.pganalyze.com/${distribution}/${version}/ stable main"
  fi
  curl -s -L https://packages.pganalyze.com/pganalyze_signing_key.asc | $maybe_sudo apt-key add -
  echo "$apt_source" | $maybe_sudo tee /etc/apt/sources.list.d/pganalyze_collector.list
  $maybe_sudo apt-get $apt_opts update <$user_input
  $maybe_sudo apt-get $apt_opts install pganalyze-collector <$user_input
else
  fail "unrecognized package kind: $pkg"
fi

if [ -n "$PGA_API_BASE_URL" ];
then
  $maybe_sudo sed -i "/^\[pganalyze\]$/a api_base_url = ${PGA_API_BASE_URL}" /etc/pganalyze-collector.conf
fi

if [ -n "$PGA_API_KEY" ];
then
  $maybe_sudo sed -i "s/^#api_key = your_api_key$/api_key = ${PGA_API_KEY}/" /etc/pganalyze-collector.conf
fi

echo "Checking install by running 'pganalyze-collector --version'"
pganalyze-collector --version
echo

echo "The pganalyze collector was installed successfully"
echo

if [ -n "$PGA_GUIDED_SETUP" ];
then
  # We want all opts passed separately here: not sure why this is not an issue above for apt-get and yum
  # shellcheck disable=SC2086
  $maybe_sudo pganalyze-collector-setup $pgags_opts <$user_input
else
  echo "Please continue with setup instructions in-app or at https://pganalyze.com/docs/install"
fi
