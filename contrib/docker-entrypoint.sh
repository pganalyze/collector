#!/bin/sh

set -e

if [ -n "$CONFIG_CONTENTS" ]; then
  echo "$CONFIG_CONTENTS" > /home/pganalyze/.pganalyze_collector.conf
fi

CMD_PREFIX=exec
if [ $(id -u) = 0 ]; then
  CMD_PREFIX="exec setpriv --reuid=pganalyze --regid=pganalyze --inh-caps=-all --clear-groups"
fi

if [ "$1" = 'test' ]; then
  eval $CMD_PREFIX /home/pganalyze/collector --test --no-log-timestamps
fi

if [ "$1" = 'test-explain' ]; then
  eval $CMD_PREFIX /home/pganalyze/collector --test-explain --no-log-timestamps
fi

if [ "$1" = 'collector' ]; then
  eval $CMD_PREFIX /home/pganalyze/collector --statefile=/state/pganalyze-collector.state --no-log-timestamps
fi

eval $CMD_PREFIX "$@"
