#!/bin/sh

set -e

if [ "$1" = 'test' ]; then
  exec /usr/local/bin/gosu pganalyze /home/pganalyze/collector --test --no-log-timestamps
fi

if [ "$1" = 'collector' ]; then
  exec /usr/local/bin/gosu pganalyze /home/pganalyze/collector --statefile=/state/pganalyze-collector.state --no-log-timestamps
fi

exec /usr/local/bin/gosu pganalyze "$@"
