perl -pi -e "s/#shared_preload_libraries = ''/shared_preload_libraries = 'pg_stat_statements'/g" \
/var/lib/postgresql/data/postgresql.conf
