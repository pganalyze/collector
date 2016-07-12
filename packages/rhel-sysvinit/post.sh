#!/bin/sh

set -e

if [ "$1" -eq "1" ]; then
  # First version of the package being installed
  chkconfig --add pganalyze-collector
  chkconfig pganalyze-collector on
  service pganalyze-collector start
elif [ "$1" -eq "2" ]; then
  # Upgrade
  service pganalyze-collector restart
fi
