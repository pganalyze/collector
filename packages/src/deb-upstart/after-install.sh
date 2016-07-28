if ! getent passwd pganalyze > /dev/null; then
  adduser --system --quiet --home /var/lib/pganalyze-collector --no-create-home --shell /bin/bash pganalyze
fi
mkdir -p /var/lib/pganalyze-collector
su -s /bin/sh pganalyze -c "test -O /var/lib/pganalyze-collector" || chown pganalyze /var/lib/pganalyze-collector

start -q pganalyze-collector
