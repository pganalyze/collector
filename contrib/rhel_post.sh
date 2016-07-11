#!/bin/sh

set -e

chkconfig --add pganalyze-collector
chkconfig pganalyze-collector on
service pganalyze-collector start
