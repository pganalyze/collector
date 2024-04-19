#!/bin/sh

set -e

if [ -n "$CONFIG_CONTENTS" ]; then
  echo "$CONFIG_CONTENTS" > /config/pganalyze-collector.conf
fi

CMD_PREFIX=exec
if [ $(id -u) = 0 ]; then
  CMD_PREFIX="exec setpriv --reuid=pganalyze --regid=pganalyze --inh-caps=-all --clear-groups"
fi

if [ "$1" = 'test' ]; then
  shift
  eval $CMD_PREFIX /home/pganalyze/collector --config=/config/pganalyze-collector.conf --test --no-log-timestamps --no-reload "$@"
elif [ "$1" = 'test-explain' ]; then
  shift
  eval $CMD_PREFIX /home/pganalyze/collector --config=/config/pganalyze-collector.conf --test-explain --no-log-timestamps "$@"
elif  [ "$1" = 'collector' ]; then
  shift
  eval $CMD_PREFIX /home/pganalyze/collector --config=/config/pganalyze-collector.conf --statefile=/state/pganalyze-collector.state --no-log-timestamps "$@"
else
  eval $CMD_PREFIX "$@"
fi
