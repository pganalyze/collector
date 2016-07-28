#!/bin/bash

set -e

# Verify that we're running after package installation
service pganalyze-collector status | grep -q running

# Verify that we're running as a non-priviledged user
ps u -U pganalyze | grep -q pganalyze-collector

# Verify that reloading works and emits a log notice
service pganalyze-collector reload
tail /var/log/messages | grep -q "Reloading configuration"

# Verify that stopping works
service pganalyze-collector stop

echo "Test successful"
