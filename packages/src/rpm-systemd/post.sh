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

if [ "$1" -eq "1" ]; then
  # First version of the package being installed
  systemctl daemon-reload
  systemctl start pganalyze-collector.service
elif [ "$1" -eq "2" ]; then
  # Upgrade
  systemctl stop pganalyze-collector.service
  systemctl start pganalyze-collector.service
fi
