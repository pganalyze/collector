#!/bin/sh

if grep -q citus /var/lib/postgresql/data/postgresql.conf;
then
  perl -pi -e "s/shared_preload_libraries='citus'/shared_preload_libraries = 'citus,pg_stat_statements'/g" /var/lib/postgresql/data/postgresql.conf
else
  perl -pi -e "s/#shared_preload_libraries = ''/shared_preload_libraries = 'pg_stat_statements'/g" /var/lib/postgresql/data/postgresql.conf
fi

echo "Enabled pg_stat_statements"
