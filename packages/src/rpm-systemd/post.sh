#!/bin/sh

set -e

# Always ensure the user exists as intended
if ! getent passwd pganalyze > /dev/null; then
  adduser --system --home /var/lib/pganalyze-collector --no-create-home --shell /bin/bash pganalyze
fi
mkdir -p /var/lib/pganalyze-collector
su -s /bin/sh pganalyze -c "test -O /var/lib/pganalyze-collector" || chown pganalyze /var/lib/pganalyze-collector

if [ "$1" -eq "1" ]; then
  # First version of the package being installed
  systemctl daemon-reload
  systemctl start pganalyze-collector.service
elif [ "$1" -eq "2" ]; then
  # Upgrade
  systemctl stop pganalyze-collector.service
  systemctl start pganalyze-collector.service
fi
