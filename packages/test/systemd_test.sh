#!/bin/bash

set -x
set -e

# Verify that we're running after package installation
systemctl status pganalyze-collector | grep -q running

# Verify that we're running as a non-priviledged user
ps u -U pganalyze | grep -q pganalyze-collector

# Give collector time to do initial start up and be able to handle SIGHUP
# (this can be slow when emulating a different architecture with QEMU)
sleep 3

# Verify that reloading works and emits a log notice
systemctl reload pganalyze-collector
journalctl -u pganalyze-collector | grep -q "Reloading configuration"

# Verify that stopping works
systemctl stop pganalyze-collector

echo "Test successful"
