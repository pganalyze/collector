#!/bin/bash

set -e

# Verify that we're running after package installation
status pganalyze-collector | grep -q running

# Verify that we're running as a non-priviledged user
ps u -U pganalyze | grep -q pganalyze-collector

# Verify that reloading works and emits a log notice
reload pganalyze-collector
tail /var/log/syslog | grep -q "Reloading configuration"

# Verify that stopping works
stop pganalyze-collector

echo "Test successful"
