# Grafana Cloud DB Metrics

This app keeps the existing pganalyze collector web dyno unchanged and adds a second Heroku process type for Grafana Cloud database metrics.

## Process Types

```Procfile
web: collector --statefile=/app/state.out --no-log-timestamps
grafana-db-metrics: /app/.apt/usr/bin/alloy run /app/config/grafana-db-metrics.alloy --storage.path=/tmp/alloy
```

The `grafana-db-metrics` dyno runs Grafana Alloy in the same Heroku app and space as the pganalyze collector, so it can reach the private Heroku Postgres database and push metrics outbound to Grafana Cloud.

## Buildpack Requirement

The app must use the Heroku Apt buildpack before the Go buildpack:

```bash
heroku buildpacks:add --index 1 heroku-community/apt --app <collector-app-name>
```

`Aptfile` installs Grafana Alloy from the official GitHub release package. Heroku installs Apt buildpack packages under `/app/.apt`, so the process command uses `/app/.apt/usr/bin/alloy`.

## Required Config Vars

Set these on the collector app:

```bash
heroku config:set \
  GRAFANA_POSTGRES_URL='<readonly-postgres-url>' \
  PROMETHEUS_REMOTE_WRITE_URL='<grafana-prometheus-remote-write-url>' \
  PROMETHEUS_USERNAME='<grafana-prometheus-instance-id>' \
  GRAFANA_CLOUD_TOKEN='<grafana-cloud-token-with-metrics-write>' \
  GRAFANA_ENVIRONMENT='production' \
  --app <collector-app-name>
```

Use a Grafana Cloud Access Policy token with:

```text
metrics:write
```

Prefer a read-only Postgres user for `GRAFANA_POSTGRES_URL`. If you reuse an existing Heroku Postgres URL, confirm it does not grant application write privileges unless that risk is accepted.

## Deploy

Deploy the app normally, then scale the metrics dyno:

```bash
heroku ps:scale grafana-db-metrics=1 --app <collector-app-name>
```

Keep the pganalyze web dyno as-is:

```bash
heroku ps:scale web=1 --app <collector-app-name>
```

## Verify

Watch the metrics dyno:

```bash
heroku logs --tail --app <collector-app-name> --dyno grafana-db-metrics
```

In Grafana Explore, use the Prometheus/Metrics datasource and query:

```promql
pg_up{space="accel-sorcery-apps"}
```

If no data appears, broaden the query:

```promql
{space="accel-sorcery-apps"}
```
