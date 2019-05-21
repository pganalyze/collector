#!/bin/sh

set -e

# Always ensure the user exists as intended
if ! getent passwd pganalyze > /dev/null; then
  adduser --system --home /var/lib/pganalyze-collector --no-create-home --shell /bin/bash pganalyze
fi

if ! getent group pganalyze > /dev/null; then
  addgroup --system --quiet pganalyze
  usermod -g pganalyze pganalyze
fi

mkdir -p /var/lib/pganalyze-collector
su -s /bin/sh pganalyze -c "test -O /var/lib/pganalyze-collector" || chown pganalyze /var/lib/pganalyze-collector

chown root:pganalyze /usr/bin/pganalyze-collector-helper
chmod 4750 /usr/bin/pganalyze-collector-helper

chown root:pganalyze /etc/pganalyze-collector.conf
chmod 640 /etc/pganalyze-collector.conf

if [ "$1" -eq "1" ]; then
  # First version of the package being installed
  chkconfig --add pganalyze-collector
  chkconfig pganalyze-collector on
  service pganalyze-collector start
elif [ "$1" -eq "2" ]; then
  # Upgrade
  service pganalyze-collector restart
fi
