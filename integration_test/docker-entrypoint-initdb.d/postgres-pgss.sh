#!/bin/sh

if grep -q citus "$PGDATA"/postgresql.conf;
then
  perl -pi -e "s/shared_preload_libraries='citus'/shared_preload_libraries = 'citus,pg_stat_statements'/g" "$PGDATA"/postgresql.conf
else
  perl -pi -e "s/#shared_preload_libraries = ''/shared_preload_libraries = 'pg_stat_statements'/g" "$PGDATA"/postgresql.conf
fi

echo "Enabled pg_stat_statements"
