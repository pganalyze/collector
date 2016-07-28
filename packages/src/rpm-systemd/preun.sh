#!/bin/sh

set -e

if [ "$1" -eq "0" ]; then
  # Last version of the package is being erased
  systemctl stop pganalyze-collector.service
fi
