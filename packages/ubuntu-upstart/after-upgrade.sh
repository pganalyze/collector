#!/bin/sh

set -e

if [ status pganalyze-collector | grep -q running ]; then
  restart pganalyze-collector
else
  start pganalyze-collector
fi
