# Changelog

## 0.66.2      2025-06-25

* Amazon Aurora: Add support for Postgres 17
  - The collector now supports Amazon Aurora with Postgres 17
  - Previously, Amazon Aurora users on Postgres 17 that also had plan statistics
    enabled were unable to collect query statistics due to a column "blk_read_time"
    does not exist error
* Log Insights: Improve parsing Heroku auto_explain logs using JSON format
  - Support newlines in the middle of the EXPLAIN query with the JSON format


## 0.66.1      2025-05-30

* Add support for Google AlloyDB IAM connections
  - This allows monitoring of Google AlloyDB databases using the existing
    `db_use_iam_auth` / `DB_USE_IAM_AUTH` setting: If enabled, the collector
    fetches a short-lived token for logging into the database instance from the
    GCP API, instead of using a hardcoded password in the collector
    configuration file
  - To set this up, see our [updated Google AlloyDB setup
    documentation](https://pganalyze.com/docs/install/google_cloud_sql/01_create_monitoring_user)


## 0.66.0      2025-05-07

* Collect statistics from pg_stat_io (Postgres 16+)
* Amazon Aurora: Pick up the correct oldest xmin for Aurora replication
  - Aurora does not include the correct replication details in
    pg_stat_replication, so check aurora_replica_status and use the older
    of the two values to track XminHorizonStandby on the platform.
* Amazon RDS: Also check monitoring interval for Enhanced Monitoring
* Google Cloud SQL: Assume Private IP for IAM authentication-based connections
  - If the legacy behavior (which connects over Public IP) is required, the new
    `gcp_use_public_ip` / `GCP_USE_PUBLIC_IP` setting makes the collector
    connect over Public IP instead
* Crunchy Bridge: Fetch all system metrics via API
  - The collector previously collected some metrics from the container itself,
    but this has shown to be unreliable due to recent provider changes.
  - Instead, retrieve a more limited set of metrics from the official APIs
    provided by Crunchy Bridge. Setting an API key per the documentation
    is required for this to work.
* Kubernetes: Add ability to use raw environment variables maps in deployment
* Better support for running EXPLAIN with pganalyze.explain_analyze helper
* Improve warning when collector is paused due to a duplicate already running
* Improve error messages when the monitoring user is not properly set up
* Remove --no-postgres-(functions|bloat|views) flags to collector
  - These flags do not have any effect and are very unlikely to be in use.


## 0.65.0      2025-02-20

* Google Cloud SQL: Support IAM Authentication
  - This allows monitoring of Google Cloud SQL databases using the existing
    `db_use_iam_auth` / `DB_USE_IAM_AUTH` setting: If enabled, the collector
    fetches a short-lived token for logging into the database instance from the
    GCP API, instead of using a hardcoded password in the collector
    configuration file
  - To set this up, see our [updated Google Cloud SQL setup
    documentation](https://pganalyze.com/docs/install/google_cloud_sql/01_create_monitoring_user)
* Start collecting and report pg_stat_statements_info stats
  - This makes it easier to track down some pg_stat_statements-related
    problems
* Support monitoring non-Heroku databases with Heroku-hosted collector
* Follow symlinks when tracking storage stats for data directory
  - This fixes storage statistics accounting for data directories symlinked
    to other partitions
* Fix pg_hint_plan handling for collector-driven Query Tuning workflow
* Fix errors when collecting stats about frequently-locked partitions
* Fix receiving logs through syslog
  - This was inadvertently broken by the log parsing changes in v0.58.0
* Avoid log parsing error when database connection cannot be established
  - This was only a secondary problem, but the stack trace in the logs could
    make it harder to track down the root cause
* Sign built Go binaries for macOS
  - Due to security enhancements in newer macOS versions, unsigned Go
    binaries may hang when built and executed locally; signing makes it
    easier to debug the collector on macOS
* Update Go version to 1.23
* Update Dockerfile alpine base image to 3.21

## 0.64.1      2025-01-08

* Fix database connection leak in buffercache logic
  - This was previously not an issue since connections auto-closed after 30 seconds,
    but due to a connection handling change in 0.64.0, this broke and caused
    pganalyze user connection limits to be hit


## 0.64.0      2025-01-07

* Support for pganalyze Query Tuning Workbooks
  - The collector now optionally executes on-demand EXPLAIN ANALYZE queries for
    the new Query Tuning feature via the new pganalyze.explain_analyze() helper
  - This helper is owned by a separate user which gets assigned table read
    permissions, and avoids granting the collector user unnecessary permissions
    directly
  - By creating the helper function you opt into automated query runs through
    the collector. For high security environments, Query Tuning Workbooks can
    still be used without this feature by running queries manually
  - For easily creating the helper in all databases on a server the
    new "--generate-explain-analyze-helper-sql" command is added
  - The enable_query_runner setting introduced in 0.63.0 is removed,
    since the helper function is now mandatory to use this feature
* Update pg_query to v6 / Postgres 17 parser
* Install script: Add AlmaLinux and Rocky Linux support
* PII filtering bug fixes
  - Correctly handle secondary lines that were not analyzed
  - Detect bind parameters in CONTEXT as statement_parameter
* Other bug fixes
  - Azure: Improve system metrics error handling, and correctly return most recent value
  - Add missing MAX_BUFFER_CACHE_MONITORING_GB configuration variable
  - Track parent partition size when child partitions are untracked
  - Store zero-value table stats when diff doesn't exist
  - DB connections: Don't limit to 30 seconds lifetime to avoid timeout bugs
* Routine security updates
  - Bump golang.org/x/crypto from 0.25.0 to 0.31.0
  - Update golang.org/x/net to v0.33.0


## 0.63.0      2024-11-21

* Fix WebSocket error handling
  - If the WebSocket mechanism hit an error at the wrong time, this could cause
    a stuck collector state, where the collector would keep running but stop
    processing and sending snapshots
* Track Postgres buffer cache usage
  - This reports statistics from [pg_buffercache](https://www.postgresql.org/docs/current/pgbuffercache.html)
    if available
  - Since this can be slow, and grows slower with larger buffer size, this can be
    configured with the new setting `max_buffer_cache_monitoring_gb` (default 200 GB)
* Fix partitioned table stats handling
  - Partitioned table stats are now reported as aggregations over child partition stats
* Add collector query runner
  - This provides a mechanism for the collector to help pganalyze users run
    EXPLAIN queries in future versions of pganalyze
  - This is disabled by default
* Update packaging scripts to use the `groupadd` command instead of `addgroup` when installing
  - `addgroup` is not available on some newer distributions, e.g., Amazon Linux 2023


## 0.62.0      2024-11-13

* Fix PII filtering for detail log lines
  - Due to a bug in 0.60.0, all detail log lines (lines that add additional
    context to the primary log message) were unnecessarily redacted
* Add DB_URL_FILE and DB_PASSWORD_FILE ([@Munksgaard](https://github.com/Munksgaard))
  - This allows passing sensitive DB passwords through files instead of environment
    variables. This makes collector work better with [systemd
    credentials](https://systemd.io/CREDENTIALS/) and NixOS flakes.
* Collect query plan information on Amazon Aurora
  - This collects query plans and statistics in full snapshots using the [aurora_stat_plans](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/aurora_stat_plans.html) function.
* Update systemd file to use MemoryMax instead of MemoryLimit
  - The latter is [deprecated](https://www.freedesktop.org/software/systemd/man/latest/systemd.resource-control.html#MemoryMax=bytes)


## 0.61.0      2024-10-23

* Store query texts in temporary file before fingerprinting them
  - This fixes an increase in reported runtime of collector queries,
    due to a slow loading of the result set introduced in release 0.60.0
* Log Insights: Use more specific log parsing regexp
  - This avoids incorrectly sending application/database/role identifiers longer
    than 63 characters (Postgres' built-in limit) when there are parsing issues
* Show test error when pg_stat_statements version is below 1.9 with Postgres 14+
  - Old pg_stat_statements extension schemas don't correctly include the
    "toplevel" attribute, and can cause bogus query statistics when there
    is a mismatch between the extension schema and the shared library code
* Track cluster identifier (cluster ID) as part of system information
* Keep fetching column stats with outdated helper function on Postgres < 17


## 0.60.0      2024-10-15

WARNING: For Enterprise Server releases older than 2024.10 using [a separate collector installation](https://pganalyze.com/docs/enterprise/setup/separate-collector-install), this release is partially incompatible. Log Insights will not receive any data if using this collector version. Enterprise Server installations using an integrated collector are not affected, nor are Scale and Production plans.

* Update `get_column_stats` for Postgres 17
  - The function signature has changed, so must be dropped and recreated
    https://pganalyze.com/docs/install/troubleshooting/column_stats_helper
* Handle ambiguous log lines more reliably
  - This fixes parsing of some log lines for Google CloudSQL
* Change log upload to include log text directly in snapshots, instead of uploading it separately to S3
  - This simplifies the Enterprise Server setup by making object storage optional
* Reduce memory usage when processing `pg_stat_statements`
* Remove temp file usage from collector


## 0.59.0      2024-10-01

* Use new WebSocket-based API for snapshot submissions
  - Long-lived WebSocket connections have lower overhead for individual snapshots
    that send statistics data to pganalyze, and avoid repeated HTTP connections
  - In case of errors when connecting the collector will fall back to regular
    HTTP-based snapshots and emit a warning (e.g. due to a misconfigured proxy, or when
    connecting to a pganalyze Enterprise Server install without WebSocket support)
* Automated EXPLAIN (auto_explain) improvements
  - Unless filtered, keep query parameters included with auto_explain as part of query samples
  - Improve handling of newlines with auto_explain "text" format
* Azure Database / Cosmos DB for PostgreSQL: Collect system info and metrics
  - To start using this, you need to supply a new config variable AZURE_SUBSCRIPTION_ID to the collector, as well as setting up managed identity (like is done for Log Insights support)
  - The managed identity now additionally needs access to the Monitoring Reader role on the Azure Database instance
* Improve support for EDB Postgres Advanced Server
* Postgres 17: Update pg_stat_progress_vacuum field names
* Log Insights: Complete transition to new log parser (introduced in 0.58.0)
  - Drop supported log_line_prefix check in test
  - Drop legacy log line parsing mechanism
  - Fix --analyze-logfile flag


## 0.58.0      2024-08-30

* Log Insights: Revamp log parsing mechanism
  - The new mechanism is more performant and allows for arbitrary
    log_line_prefix settings. The new parsing mechanism is the default, but you
    can set `db_log_line_prefix = legacy` in the config file or
    `LOG_LINE_PREFIX=legacy` in the environment to revert to the old mechanism.
* Log Insights: Redact parameters from utility statements by default
  - Statements like `CREATE USER u WITH PASSWORD 'passw0rd'` can leak sensitive
    data into Log Insights, so they are now redacted by default. Note that these
    statements are usually very fast, and are normally only logged in edge cases,
    like a lock wait problem relating to the statement.
* RDS: Update AWS SDK to v1.55.3
  - This allows using EKS pod identity; documentation coming soon
* Azure:
  - Ensure correct log handling for all Flexible Server events (don't accidentally treat them as Single Server events)
  - Support log parsing for Azure Database for Cosmos DB Postgres
* Crunchy Bridge:
  - Fix error handling for error responses from Crunchy Bridge API
* Fix hang on exit with the `--discover-log-location` flag


## 0.57.1      2024-07-17

* Log Insights:
  - Fix handling of syntax error events when STATEMENT is missing
  - Support multi-line logs with AlloyDB
* Test run: Improve handling of interrupts via CTRL+C (SIGINT)
  - Avoid collector hanging, and don't print summary
  - Allow HTTP clients to be cancelled to avoid shutdown delays
* Add option to avoid collecting distributed index stats for Citus
  - This allows setting the `DISABLE_CITUS_SCHEMA_STATS` / `disable_citus_schema_stats`
    setting to the "index" value, which will cause the collector to
    skip collecting index statistics for Citus distributed tables
    (which can time out when there is a significant count of indexes)
* Install script: Avoid deprecated usage of apt-key command


## 0.57.0      2024-06-19

* Log Insights: Add support for receiving logs via OpenTelemetry
  - The collector can now start a built-in OTLP HTTP server that receives logs
    at a specified local address via `db_log_otel_server` / `LOG_OTEL_SERVER`
  - This can be used with self-managed servers running in a Kubernetes cluster,
    combined with a telemetry agent like Fluent Bit
* Exclude internal Postgres tables from stats helper functions
  - With Amazon RDS/Aurora, stats collection could fail with "permission denied
    for attribute pg_subscription.subconninfo"
  - Update stats helper functions to explicitly exclude references causing this
    issue
* Log Insights: Improve parsing with Heroku auto_explain logs
  - With auto_explain logs of Heroku Postgres, new lines in the middle of the
    EXPLAIN query are observed, which has been preventing the log parser from
    correctly handling these EXPLAIN queries
  - Add a workaround to mitigate this issue when such unexpected new lines are
    detected
* Enable log filtering by default to avoid storing database secrets
  - `filter_log_secret` now defaults to `credential,parsing_error,unidentified`
* Improve log filtering for syntax errors
  - Previously when `filter_log_secret: syntax_error` is set, the full statement
    would still be included in the logs. It's now properly redacted.
* AWS: Allow setting both assume role and web identity/role ARN
  - Previously when both of them are set, web identity/role ARN were ignored
  - With the change, we now first retrieve credentials via web identity, and
    then assume the role specified as `aws_assume_role` / `AWS_ASSUME_ROLE`
  - This helps with cross-account configurations on AWS in combination with the
    collector running in EKS
* Add packages for Ubuntu 24.04
* Remove "report" functionality
  - This has long been deprecated. Removing the code as a cleanup
* Stop building packages for CentOS 7 / RHEL 7
  * CentOS 7 / RHEL 7 is end of life. The minimum required glibc version for RPM packages is
    now 2.26 (e.g. Amazon Linux 2)


## 0.56.0      2024-04-19

* Improve the collector test output
  - In addition to the existing test output, the new summary is added to provide
    a consolidated result showing the state of the collector setup
  - Add more verbose output for the `--test-explain` flag
* Amazon RDS/Aurora: Use 5432 as a default DB port
  - Previously IAM authentication would fail with "PAM authentication failed"
    when the port was not explicitly set in the collector configuration
* Update pg_stat_statements logic
  - Support updated fields in Postgres 17
* Autovacuum:
  - Add support for updated log format (frozen:) in Postgres 16+
* Publish Helm Chart package
  - The Helm Chart repository can be accessed via https://charts.pganalyze.com/
  - The collector chart is available at `pganalyze/pganalyze-collector`
  - The oldest available package version is 0.55.0
* Docker image: Support taking additional arguments for `test`, `test-explain`, `collector`
  - Previously, adding the verbose flag like `test -v` wasn't working. With this
    update, the additional arguments are now correctly passed to the process and
    `test -v` will run the test with verbose mode
* Docker image: Update the internal collector config file location
  - When the Docker container is passed the `CONFIG_CONTENTS` environment variable,
    the file used to be written to `/home/pganalyze/.pganalyze_collector.conf`
    location, and then read by the collector
  - Instead, this file is now written to the `/config/pganalyze-collector.conf`
    location - this fixes problems when having a read-only root filesystem
* Add `--generate-stats-helper-sql` helper command
  - This command generates a SQL script that can be passed to `psql` to install
    stats helpers (e.g. for collecting column stats) on all configured databases
    for the specified server


## 0.55.0      2024-03-27

* Add integration with Tembo
  - Supports Log Insights (via log streaming) and system metrics download
  - This integration is mainly intended for direct use by the Tembo Postgres
    provider (the collector is deployed by Tembo, if enabled)
* Heroku integration
  - Avoid unnecessary error messages related to state file and reload mechanism
* Accept PGA_API_BASE_URL env var in addition to PGA_API_BASEURL
  - Going forward we recommend using `PGA_API_BASE_URL` when configuring the
    collector for sending to pganalyze Enterprise Server installations
* Syslog handler: Allow leading spaces before parts regexp
  - When configuring rsyslogd for RFC5424 output with the
    RSYSLOG_SyslogProtocol23Format template, it adds a leading space that we
    didn't anticipate correctly.
* Relation stats: Call pg_stat_get_* directly instead of using system views
  - The collecror now calls the underlying pg_stat_get* functions directly,
    which has the same effect as querying the pg_stat_all_tables and
    pg_statio_all_tables views (as they are simple views without any security
    barrier), but results in better performance when a table filter
    (`ignore_schema_regexp` / `IGNORE_SCHEMA_REGEXP`) is active


## 0.54.0      2024-02-23

* Update pg_query_go to v5 / Postgres 16 parser
* Bugfix: Skip collecting extended statistics for Postgres 11 and below
  - Since the system view `pg_stats_ext` was introduced starting with Postgres
    12, this was causing the issue of collecting any schema data on Postgres 11
    and below


## 0.53.0      2024-02-02

* Track extended statistics created with `CREATE STATISTICS`
  - This is utilized by pganalyze Index Advisor to better detect functional dependencies, and improve multi-column index recommendations
  - To allow the collector to access extended statistics data you need to create the new "get_relation_stats_ext" helper function (see https://pganalyze.com/docs/install/troubleshooting/ext_stats_helper)
* Docker image: Don't reload when calling "test" command


## 0.52.4      2023-12-21

* Log Insights: Add support for receiving syslog over TLS
  - You can configure a TLS certificate for the collector syslog server using
    the following config settings:
    - `db_log_syslog_server_cert_file` / `LOG_SYSLOG_SERVER_CERT_FILE` or
      `db_log_syslog_server_cert_contents` / `LOG_SYSLOG_SERVER_CERT_CONTENTS`
    - `db_log_syslog_server_key_file` / `LOG_SYSLOG_SERVER_KEY_FILE` or
      `db_log_syslog_server_key_contents` / `LOG_SYSLOG_SERVER_KEY_CONTENTS`
    - The Certificate Authority both on the server side and the client side also
      can be specified via config settings
* Azure: Fix managed identity credential creation in Log Insights
  - This fixes a failure of obtaining logs from Azure when the managed identity
    credential was used. This was with the "failed to set up workload identity
    Azure credentials" error message
* Citus: Avoid error collecting schema stats on tables with no indexes


## 0.52.3      2023-11-30

* Collector log output: Reduce frequency of some snapshot log events
  - Previously, near real-time "compact" snapshots would generate log lines
    every 10 seconds, which made errors hard to find
  - Now, a single log line is printed once a minute with a summary of snapshots
    submitted
  - Note that `--verbose` will still log every snapshot as it's submitted
* Collector log output: Add "full" prefix for full snapshots sent every 10 minutes
  - This changes the "Submitted snapshot successfully" message to read
    "Submitted full snapshot successfully" instead
* OpenTelemetry integration:
  - Support `pganalyze` tracestate to set start time of the span
  - Start time can be specified with `t` member key as Unix time in seconds,
    with decimals to specify precision down to nano seconds
  - This allows specifying a better span start and end time in case precise
    timestamps are not present in the Postgres logs, like with Amazon RDS
* Allow pg_stat_statements failures and continue snapshot processing
  - Previously, when pg_stat_statements data collection failed (e.g. a timeout
    when the query text file got too large), the whole snapshot was treated as
    failed and only reported an error snapshot to pganalyze, without any
    statistics
  - Instead, treat pg_stat_statements errors as a collector error in the
    snapshot, but continue afterwards and report other statistics that were
    collected successfully

## 0.52.2      2023-10-26

* OpenTelemetry integration:
  - Support sqlcommenter format query tag (`key='value'`) for `traceparent`
  - Add a new config setting `otel_service_name` / `OTEL_SERVICE_NAME` for
    customizing the OpenTelemetry service name


## 0.52.1      2023-10-11

* Postgres 14+: Include toplevel attribute in statement statistics key
  - This could have caused statistics to be incorrect in Postgres 14+ when
    the same query was called both from within a function (toplevel=false)
    and directly (toplevel=true), with pg_stat_statements.track set to "all"
  - If affected, the issue may have shown by bogus statistics being recorded,
    for example very high call counts, since the statement stats diff would
    not have used the correct reference


## 0.52.0      2023-10-04

* OpenTelemetry integration: Allow exporting EXPLAIN plans as trace spans
  - This is an experimental feature that allows configuring the collector
    to send an OpenTelemetry tracing span for each processed EXPLAIN plan
    with an associated `traceparent` query tag (e.g. set by sqlcommenter)
    to the configured OpenTelemetry endpoint
  - To configure the OTLP protocol endpoint, set the new config setting
    `otel_exporter_otlp_endpoint` / `OTEL_EXPORTER_OTLP_ENDPOINT`, with a
    endpoint string like "http://localhost:4318". You can also optionally
    set the `otel_exporter_otlp_headers` / `OTEL_EXPORTER_OTLP_HEADERS`
    variable to add authentication details used by hosted tracing providers
    like Honeycomb and New Relic
* Relax locking requirements for collecting table stats
  - This avoids skipped statistics due to page or tuple level locks,
    which do not conflict with `pg_relation_size` as run by the collector.
* Activity snapshots: Normalize queries for `filter_query_sample = normalize`
  - This matches the existing behavior when `filter_query_sample` is set
    to `all`, which is to run the normalization function on pg_stat_activity
    query texts, making sure all parameter values are replaced with `$n`
    parameter references
* Self-managed servers: Add test run notice when system stats are skipped
* Docker log tail: Re-order args to also support podman aliased as docker


## 0.51.1      2023-08-15

* Fix handling of tables that only have an entry in pg_class, but not pg_stat_user_tables
  - Due to a bug introduced in the last release (0.51.0), databases with such tables would
    error out and be ignored due to n_mod_since_analyze and n_ins_since_vacuum being NULL


## 0.51.0      2023-08-12

* Autovacuum:
  - Add support for updated log format in Postgres 15+
  - Remember unqualified name for "skipping vacuum" log events
  - Add more cases for "canceling autovacuum task" log context line
  - Track n_ins_since_vacuum value to determine when insert-based autovacuum was triggered
* AWS Aurora: Correctly detect Aurora reader instances as replicas
* Self-managed servers: Use log_timezone setting to determine log timezone if possible
* Azure: Fix partition selection issue in Azure log processing
* Helm chart: Improve default security settings
* Update Go version to 1.21
* Packages:
  - Switch to SHA256 signatures to fix RHEL9 install errors
  - Drop Ubuntu 16.04, 18.04 and Debian 10 (Buster) support, as they are no longer supported


## 0.50.1      2023-06-29

* Bugfix: Return correct exit code with the data collection test run
  - The correct exit code was returned with "--reload --test", but not with "--test"
* Xmin horizon metrics: Fix incorrect ReplicationSlotCatalog
  - ReplicationSlot was wrongly sent as ReplicationSlotCatalog
  - Xmin horizon metrics collection was introduced in 0.49.0
* Update github.com/satori/go.uuid to 1.2.0
  - Fixes CVE-2021-3538 which may have led to random UUIDs having less
    randomness than intended
  - Effective security impact of this historic issue is expected to be minimal,
    since random UUIDs are only used for snapshot identifiers associated to a
    particular pganalyze server ID
* Log Insights: Add autovacuum index statistics information introduced in Postgres 14
  - Previously, if autovacuum logs included such information, the collector
    failed to match the log line and the events would not be classified
    correctly in Log Insights


## 0.50.0      2023-06-05

* Track TOAST table name, reltuples and relpages
* Reload collector config after successful test run
  - If you have previously run "--reload --test" you can now simply run "--test"
  - To restore the old behaviour, pass the "--no-reload" flag together with "--test"
* RPM packages: Ensure collector gets started after reboot
  - Due to a packaging oversight, the pganalyze-collector service was not correctly
    enabled in systemd, which caused the collector to not start after a system reboot
  - If you are upgrading, the package upgrade script will automatically fix this for you
* Collector install script: Add Amazon Linux 2023, refresh other versions
* Azure Database for PostgreSQL / Azure Cosmos DB for PostgreSQL
  - Add support for Azure Kubernetes Service (AKS) Workload Identity
    - To utilize this integration, follow the regular Azure instructions for workload
      identity - the relevant environment variables will be automatically recognized
* Amazon RDS:
  - auto_explain and slow query log: Look for "[Your log message was truncated]" marker
    in the middle of multi-line log messages, not just at the end
    - This can occur due to limitations of the AWS API - this way the log line is
      correctly marked as truncated, instead of as a parsing error
* Heroku Postgres:
  - Rewrite syslog parsing code and inline it, to avoid "lpx" library license ambiguity


## 0.49.2      2023-03-30

* Bugfix: Ensure all relation information will be sent out even with a lock
  - This fixes a bug where we were not sending out relation information of
    relations encountered locks. Processing a snapshot missing such information
    was failing
* Allow pg_stat_statements_reset() to fail with a soft error
  - This was a hard error previously, which failed the snapshot and the snapshot
    state did not get persisted, indirectly led to a memory leak
* Add integrity checks before uploading snapshots
  - Validate some structural assumptions that cannot be enforced by protobuf
    before sending a snapshot
* Bugfix: Increase timeout to prevent data loss when monitoring many servers
  - This mitigates an issue introduced in 0.49.0

## 0.49.1      2023-03-10

* Relation queries: Correctly handle later queries encountering a lock
  - This fixes edge cases where relation metadata (e.g. which indexes exist)
    can appear and disappear from one snapshot to the next, due to locks
    held for parts of the snapshot collection
* Relation statistics: Avoid bogus data due to diffs against locked objects
  - This fixes a bug where table or index statistics can be skipped due to
    locks held on the relation, and that causing a bad data point to be
    collected on a subsequent snapshot, since the prior snapshot would be
    missing an entry for that relation. Fixed by consistently skipping
    statistics for that table/index in such situations.
* Amazon RDS / Aurora: Support new long-lived CA authorities
  - Introduces the new "rds-ca-global" option for db_sslrootcert, which is the
    recommended configuration for RDS and Aurora going forward, which encompasses
    both "rds-ca-2019-root" and all newer RDS CAs such as "rds-ca-rsa2048-g1".
  - For compatibility reasons we still support naming the "rds-ca-2019-root" CA
    explicitly, but its now just an alias for the global set.
* Citus: Add option to turn off collection of Citus schema statistics
  - For certain Citus deployments, running the relation or index size functions
    can fail or time out due to a very high number of distributed tables.
  - Adds the new option "disable_citus_schema_stats" / "DISABLE_CITUS_SCHEMA_STATS"
    to turn off the collection of these statistics. When using this option its
    recommended to instead monitor the workers directly for table and index sizes.
* Add troubleshooting HINT when creating pg_stat_statements extension fails
  - This commonly fails due to creating pg_stat_statements on the wrong database,
    see https://pganalyze.com/docs/install/troubleshooting/pg_stat_statements


## 0.49.0      2023-02-27

* Update pg_query_go to v4 / Postgres 15 parser
  - Besides supporting newer syntax like the MERGE statement, this parser
    update also drops support for "?" replacement characters found in
    pg_stat_statements output before Postgres 10
* Postgres 10 is now the minimum required version for running the collector
  - We have dropped support for 9.6 and earlier due to the parser update,
    and due to Postgres 9.6 now being End-of-Life (EOL) for over 1 year
* Enforce maximum time for each snapshot collection using deadlines
  - Sometimes individual database servers can take longer than the allocated
    interval (e.g. 10 minutes for a full snapshot), which previously lead to
    missing data for other servers monitored by the same collector process
  - The new deadline-based logic ensures that collector functions return with
    a "context deadline exceeded" error when the allocated interval is exceed,
    causing a clear error for that server, and allowing other servers to
    continue reporting their data as planned
  - As a side effect of this change, Ctrl+C (SIGINT) now works to stop a
    collector test right away, instead of waiting for the snapshot to complete
* Log Insights
  - Only consider first 1000 characters for log_line_prefix to speed up parsing
  - Clearly report errors with closing/removing temporary files
  - Improve --analyze-logfile mode for debugging log parsing
  - Amazon RDS/Aurora: Improve handling of excessively large log file portions
  - Azure DB for Postgres: Fix log line parsing for DETAIL lines
* Collect xmin horizon metrics
* Bugfixes
  - Relation info: Correctly filter out foreign tables for constraints query
  - Return zero as FullFrozenXID for replicas
  - Update Go modules flagged by dependency scanners (issues are not actually applicable)


## 0.48.0      2023-01-26

* Update to Go 1.19
* Bugfix: Ensure relfrozenxid = 0 is tracked as full frozenxid = 0 (instead of
  adding epoch prefix)
* Amazon RDS and Amazon Aurora: Support IAM token authentication
  - This adds a new configuration setting, `db_use_iam_auth` / `DB_USE_IAM_AUTH`.
    If enabled, the collector fetches a short-lived token for logging into the
    database instance from the AWS API, instead of using a hardcoded password
    in the collector configuration file
  - In order to use this setting, IAM authentication needs to be enabled on the
    database instance / cluster, and the pganalyze IAM policy needs to be
    extended to cover the "rds-db:connect" privilege for the pganalyze user:
    https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.IAMPolicy.html
* Amazon RDS: Avoid DescribeDBInstances calls for log downloads for RDS instances
  - This should reduce issues with rate limiting on this API call in some cases
* Amazon Aurora: Cache failures in DescribeDBClusters calls for up to 10 minutes
  - This reduces repeated calls to the AWS API when the cluster identifier is
    incorrect
* Log parsing: Add support for timezones specified by number, such as "-03"


## 0.47.0      2023-01-12

* Fix RDS log processing for large log file sections
  - This fixes an issue with RDS log file chunks larger than 10MB that caused
    the collector to calculate log text source offsets incorrectly and could
    lead to mangled log text in the pganalyze UI and incorrect log filtering
* Warn if some log lines will be ignored
  - Some verbose logging settings can lead to log lines being ignored by the
    collector for performance reasons: warn about this on startup
* Improve Aiven Service ID and Project ID detection from hostname
* Fix error handling when fetching stats
  - The missing checks could previously lead to incomplete snapshots, possibly
    resulting in tables or indexes temporarily disappearing in pganalyze
* Fix error handling regarding reading SSL-related config values on startup
* Ignore non-Postgres URIs in environment on Heroku ([@Preovaleo](https://github.com/Preovaleo))
* Send additional Postgres table stats
  - Send relpages, reltuples, relallvisible
* Send additional Postgres transaction metadata
  - server level (new stats): current TXID and next MXID
  - database level: age of datfrozenxid and datminmxid, also xact_commit and
    xact_rollback
  - table level: the age of relfrozenxid and relminmxid
* Send Citus distributed index sizes
* Add `always_collect_system_data` config option
  - Also configurable with the `PGA_ALWAYS_COLLECT_SYSTEM_DATA` environment
    variable
  - This is useful for package-based setups which monitor the local server by a
    non-local IP
* Update pg_query_go version to 2.2.0
* Install script: Detect aarch64 for Ubuntu/Debian package install


## 0.46.1      2022-10-21

* Fix Postgres 15 compatibility due to version check bug
  - This fixes an issue with Postgres 15 only that caused the collector to reject
    the newer pg_stat_statements version (1.10) by accident
* Add packages for Ubuntu 22.04, RHEL9-based distributions and Fedora 36


## 0.46.0      2022-10-21

* Relation stats: Skip statistics collection on child tables when parent is locked
* Add new wait events from Postgres 13 and 14
* Log streaming: Discard logs after consistent failures to upload
* Collect blocking PIDs for lock monitoring
  - Collect blocking PIDs for the backends in waiting for locks state
  - Disable this option by passing the "--no-postgres-locks" option to the collector binary
* Add "--benchmark" flag for running collector in benchmark mode (does not send data to pganalyze service)


## 0.45.2      2022-08-31

* Amazon RDS/Aurora
  - Log download: Fix edge case that caused errors on hourly log boundaries
    - Resolves errors like "Error reading 65817 bytes from tempfile: unexpected EOF"
  - Collect tags assigned to instance as system metadata
* Docker: Allow setting CONFIG_CONTENTS to pass ini-style configuration
  - This allows easier configuration of multiple servers to be monitored by
    the same Docker container. Previously this required use of a volume
    mount, which can be harder to make work successfully.
  - CONFIG_CONTENTS needs to match the regular configuration file format that
    uses separate sections for each server.
  - This can be combined with environment-variable style configuration for
    settings that apply to all servers (e.g. PGA_API_KEY) but all
    server-specific configuration should only be passed in through the
    CONFIG_CONTENTS variable.


## 0.45.1      2022-08-12

* Amazon Aurora and Amazon RDS
  - Auto-detect Aurora writer instance, as well as reader on two-node clusters
    - Previously it was required to specify the individual instance to support
      log downloads and system metrics, but this now happens automatically
    - The cluster name is auto-detected from the hostname, but to override the
      new "aws_db_cluster_id" and "aws_db_cluster_readonly" settings can be used
    - This requires giving the IAM policy for the collector the
      "DescribeDBClusters" permission
    - In case more than one reader instance is used, each reader instance must
      be specified individually instead of using the readonly cluster hostname
  - Show RDS instance role hint when running collector test
  - Ensure permission errors during log download are shown
* Add "-q" / "--quiet" flag for hiding everything except errors in the logs


## 0.45.0      2022-07-29

* Log Insights: Filter out `log_statement=all` and `log_duration=on` log lines
  - This replaces the previous behaviour that prevented all log collection for
    servers that had either `log_statement=all` or `log_duration=on` enabled.
  - With the new logic, we continue ignoring these high-frequency events
    (which would cause downstream problems), but accept all other log events,
    including threshold-based auto_explain events.
* Track extensions that are installed on each database
  - This is helpful to ensure that the necessary schema definitions are
    loaded by pganalyze, e.g. for use by the Index Advisor.
  - Ignore objects that are provided by extensions, as determined by pg_depend
    (e.g. function definitions, etc)
* Add support for Google AlloyDB for PostgreSQL
  - This adds new options to specify the AlloyDB cluster ID and instance ID
  - Special cases the log parsing to support AlloyDB's `[filename:line]` prefix
  - Supports AlloyDB's modified autovacuum log output
* Add explicit support for Aiven Postgres databases
  - Support was previously available via the self-managed instructions, but
    this adds explicit support and improved setup instructions
  - Existing Aiven servers that were detected as self-managed will be
    automatically updated to be recognized as Aiven servers
* Self-managed servers
  - Support disk statistics for software RAID devices
    - These statistics are summarized across all component disk devices and
      then tracked for the parent software RAID device as one. Note that this
      is only done in case these statistics are not yet set (which is the case
      for the typical Linux software RAID setup).
  - Allow using `pg_read_file` to read log files (instead of log tail / syslog)
    - This relies on the built-in Postgres function `pg_read_file` to read log
      files and return the log data over the Postgres connection.
    - This requires superuser (either directly or through a helper) and thus
      does not work on managed database providers, with the exception of
      Crunchy Bridge, for which this is already the mechanism to fetch logs.
    - Additionally, this carries higher overhead than directly tailing log
      files, or using syslog, and thus should only be used when necessary.
    - Set `db_log_pg_read_file = 1` / `LOG_PG_READ_FILE=1` to enable the logic
* Crunchy Bridge
  - Fix collection of system metrics
* Heroku Postgres
  - Fix blank log line parsing
* Add `--test-section` parameter to set a specific config section to test
* Fully qualify constraint definitions, to support non-standard schemas
* Add support for log_line_prefix `%m [%p] %q%u@%d ` and `%t [%p] %q%u@%d %h `


## 0.44.0      2022-06-29

* Add optional normalization of sensitive fields in EXPLAIN plans
  - Introduces new "filter_query_sample = normalize" setting that normalizes
    expression fields in the EXPLAIN plan ("Filter", "Index Cond", etc) using
    the pg_query normalization logic. Unknown EXPLAIN fields are discarded when
    this option is active.
  - Turning on this setting will also cause all query samples (whether they
    have an EXPLAIN attached or not) to have their query text normalized
    and their parameters marked as `<removed>`.
  - This setting is recommended when EXPLAIN plans may contain sensitive
    data that should not be stored. Please verify that the logic works
    as expected with your workload and log output.
  - In order to mask EXPLAIN output in the actual log stream as well (not just
    the query samples / EXPLAIN plans), make sure to use a `filter_log_secret`
    setting that includes the `statement_text` value
* Be more accepting with outdated pg_stat_statements versions
  - With this change, its no longer required to run
    "ALTER EXTENSION pg_stat_statements UPDATE" in order to use
    the collector after a Postgres upgrade
  - The collector will output an info message in case an outdated
    pg_stat_statements version is in use
* Allow pg_stat_statements to be installed in schemas other than "public"
  - This is automatically detected based on information in `pg_extension`
    and does not require any extra configuration when using a special schema
* Log Insights
  - Remove unnecessary "duration_ms" and "unparsed_explain_text" metadata
    fields, they are already contained within the query sample data
  - Always mark STATEMENT/QUERY log lines as "statement_text" log secret,
    instead of "unidentified" log secret in some cases
* Amazon RDS / Amazon Aurora
  - Fix rare bug with duplicate pg_settings values on Aurora Postgres
  - Add RDS instance role hint when NoCredentialProviders error is hit
* Heroku Postgres
  - Add support for new log_line_prefix
  - Log processing: Avoid repeating the same line over and over again
  - Fix log handling when consuming logs for multiple databases
* Google Cloud SQL
  - Re-enable log stitching for messages - whilst the GCP release notes mention
    that this is no longer a problem as of Sept 2021, log events can still be
    split up into multiple messages if they exceed a threshold around 1000-2000
    lines, or ~100kb
* Custom types: Correctly track custom type reference for array types
* Improve the "too many tables" error message to clarify possible solutions
* Fix bug related to new structured JSON logs feature (see prior release)
* Update pg_query_go to v2.1.2
  - Fixes memory leak in pg_query_fingerprint error handling
  - Fix parsing some operators with ? character (ltree / promscale extensions)


## 0.43.1      2022-05-02

* Add option for emitting collector logs as structured JSON logs ([@jschaf](https://github.com/jschaf))
  - Example output:
    ```
    {"severity":"INFO","message":"Running collector test with pganalyze-collector ...","time":"2022-04-19T12:31:05.100489-07:00"}
    ```
  - Enable this option by passing the "--json-logs" option to the collector binary
* Log Insights: Add support for Postgres 14 autovacuum and autoanalyze log events
* Column stats helper: Indicate which database is missing the helper in error message
* Azure Database for PostgreSQL
  - Add log monitoring support for Flexible Server deployment option
* Heroku Postgres
  - Fix environment parsing to support parsing of equals signs in variables
  - Log test: Don't count Heroku Postgres free tier as hard failure (emit warning instead)


## 0.43.0      2022-03-30

* Add integration for Crunchy Bridge provider
* Check citus.shard_replication_factor before querying citus_table_size
  - This fixes support for citus.shard_replication_factor > 1
* Filter out vacuum records we cannot match to a table name
  - This can occur when a manual vacuum is run in a database other than the
    primary database that is being monitored, previously leading to
    processing errors in the backend
* Docker image: Add tzdata package
  - This is required to allow timezone parsing during log line handling


## 0.42.2      2022-02-15

* Fix cleanup of temporary files used when processing logs
  - Previous collectors may have left temp files in your system's [temp directory](https://pkg.go.dev/os#TempDir)
  - To manually clean up stray temp files:
    - Shut down the collector
    - Install the new package
    - Delete any files owned by the user running the collector (pganalyze by default) in the temp directory
    - Start the collector


## 0.42.1      2022-02-01

* Log Insights
  - Handle non-UTC/non-local log_timezone values correctly
  - Use consistent 10s interval for streamed logs (instead of shorter intervals)
  - Log streams: Support processing primary and secondary lines out of order
    - This resolves issues on GCP when log lines are received out of order
  - C22 Auth failed event: Detect additional DETAIL information
  - Add regexp match for "permission denied for table" event
* Normalization: Attempt auto-fixing truncated queries
* Heroku: Do not count free memory in total memory
* Config file handling: Handle boolean values more consistently
  - Treat case-insensitive false, off, no, 'f', and 'n' as false in addition
    to zero


## 0.42.0      2021-12-20

* Provide both x86/amd64 and ARM64 packages and Docker image
  - This means you can now run the collector more easily on modern
    ARM-based platforms such as Graviton-based AWS instances
* Bugfix: Write state file before reloading collector
  - This avoids lost statistics when the collector is reloaded mid-cycle
    between the full snapshot runs
* Reduce Docker image build time and use slim image for 18x size reduction
  - With thanks to [Chris](https://github.com/dullyouth) at Kandji for this contribution


## 0.41.3      2021-12-15

 * Log Insights: Add "invalid input syntax for type json" log event
   - This is a variant of the existing invalid input syntax log event,
     with a few additional details.
 * Log Insights: Improve handling of "malformed array literal" log event
   - Add support for a double quote inside the array content
   - Mark the content as a table data log secret
   - Add the known DETAIL line "Unexpected array element"
 * Fix incorrect index recorded for unknown parent or foreign key tables
   - Previously we would sometimes use 0 in these situations, which could
     cause errors in snapshot processing
 * Heroku: drop Log Insights instructions after log test
   - This was intended to ease onboarding, but it contradicts our in-app
     instructions to wait until real snapshot data comes in to proceed
     with Log Insights setup
 * AWS: Cache RDS server IDs and errors to reduce API requests
   - This can help avoid hitting rate limits when monitoring a large number
     of servers
 * Fix issue with domains with no constraints


## 0.41.2      2021-11-18

* Add two additional log_line_prefix settings
  - `%p-%s-%c-%l-%h-%u-%d-%m `
  - `%m [%p][%b][%v][%x] %q[user=%u,db=%d,app=%a] `
* Change schema collection message to warning during test run
  - This helps discover schema collection issues, e.g. due
    to connection restrictions or other permission problems
* Fix issue with multiple domain constraints
* Upgrade gopsutil to v3.21.10
  - This adds support for M1 Macs, amongst other improvements
    for OS metris collection


## 0.41.1      2021-11-03

* Fix schema stats for databases with some custom data types
* Fix tracking of index scans over time


## 0.41.0      2021-10-14

* Add support for custom data types
* Track column stats for improved Index Advisor recommendations
* Vacuum activity: Correctly handle duplicate tables
* Citus: fix broken relation stats query
* Retry API requests in case of temporary network issues
* Update to Go 1.17
* Update to pg_query_go v2.1.0
  - Improve normalization of GROUP BY clauses


## 0.40.0      2021-06-30

* Update to pg_query_go v2.0.4
  - Normalize: Don't touch "GROUP BY 1" and "ORDER BY 1" expressions, keep original text
  - Fingerprint: Cache list item hashes to fingerprint complex queries faster
    (this change also significantly reduces memory usage for complex queries)
* Install script: Support CentOS in addition to RHEL


## 0.39.0      2021-05-31

* Docker: Use Docker's USER command to set user, to support running as non-root
  - This enables the collector container to run in environments that require the
    whole container to run as a non-root user, which previously was not the case.
  - For compatibility reasons the container can still be run as root explicitly,
    in which case the setpriv command is used to drop privileges. setpriv replaces
    gosu since its available for installation in most distributions directly, and
    fulfills the same purpose here.
* Selfhosted: Support running log discovery with non-localhost db_host settings
  - Previously this was prevented by a fixed check against localhost/127.0.0.1,
    but sometimes one wants to refer to the local server by a non-local IP address
* AWS: Add support for AssumeRoleWithWebIdentity
  - This is useful when running the collector inside EKS in order to access
    AWS resources, as recommended by AWS: https://docs.aws.amazon.com/eks/latest/userguide/specify-service-account-role.html
* Statement stats retrieval: Get all rows first, before fingerprinting queries
  - This avoids showing a bogus ClientWrite event on the Postgres server side whilst
    the collector is running the fingerprint method. There is a trade-off here,
    because we now need to retrieve all statement texts (for the full snapshot) before
    doing the fingerprint, leading to a slight increase in memory usage. Nonetheless,
    this improves debuggability, and avoids bogus statement timeout issues.
* Track additional meta information about guided setup failures
* Fix reporting of replication statistics for more than 1 follower


## 0.38.1      2021-04-02

* Update to pg_query_go 2.0.2
  - Normalize: Fix handling of two subsequent DefElems (resolves rare crashes)
* Redact primary_conninfo setting if present and readable
  - This can contain sensitive information (full connection string to the
    primary), and pganalyze does not do anything with it right now. In the
    future, we may partially redact this and use primary hostname
    information, but for now, just fully redact it.


## 0.38.0      2021-03-31

* Update to pg_query 2.0 and Postgres 13 parser
  - This is a major upgrade in terms of supported syntax (Postgres 10 to 13),
    as well as a major change in the fingerprints, which are now shorter and
    not compatible with the old format.
  - When you upgrade to this version of the collector **you will see a break
    in statistics**, that is, you will see new query entries in pganalyze after
    adopting this version of the collector.
* Amazon RDS: Support long log events beyond 2,000 lines
  - Resolves edge cases where very long EXPLAIN plans would be ignored since
    they exceeded the previous 2,000 limit
  - We now ensure that we go back up to 10 MB in the file with each log
    download that happens, with support for log events that exceed the RDS API
    page size limit of 10,000 log lines
* Self-managed: Also check for the process name "postmaster" when looking for
  Postgres PID (fixes data directory detection for RHEL-based systems)


## 0.37.1      2021-03-16

* Docker builds: Increase stack size to 2MB to prevent rare crashes
  - Alpine has a very small stack size by default (80kb) which is less than
    the default that Postgres expects (100kb). Since there is no good reason
    to reduce it to such a small amount, increase to usually common Linux
    default of 2MB stack size.
  - This would have surfaced as a hard crash of the Docker container with
    error code 137 or 139, easily confused with out of memory errors, but
    clearly distinct from it.
* Reduce timeout for accessing EC2 instance metadata service
  - Previously we were re-using our shared HTTP client, which has a rather
    high timeout (120 seconds) that causes the HTTP client to wait around
    for a long time. This is generally intentional (since it includes the
    time spent downloading a request body), but is a bad idea when running
    into EC2's IDMSv2 service that has a network-hop based limit. If that
    hop limit is exceeded, the requests just go to nowhere, causing the
    client to wait for a multiple of 120 seconds (~10 minutes were observed).
* Don't use pganalyze query marker for "--test-explain" command
  - The marker means the resulting query gets hidden from the EXPLAIN plan
    list, which is what we don't want for this test query - it's intentional
    that we can see the EXPLAIN plan we're generating for the test.


## 0.37.0      2021-02-19

* Add support for receiving logs from remote servers over syslog
  - You can now specify the new "db_log_syslog_server" config setting, or
    "LOG_SYSLOG_SERVER" environment variable in order to setup the collector
    as a syslog server that can receive logs from a remote server via syslog
    to the server that runs the collector.
  - Note that the format of this setting is "listen_address:port", and its
    recommended to use a high port number to avoid running the collector as root.
  - For example, you can specify "0.0.0.0:32514" and then send syslog messages
    to the collector's server address at port 32514.
  - Note that you need to use protocol RFC5424, with an unencrypted TCP
    connection. Due to syslog not being an authenticated protocol it is
    recommended to only use this integration over private networks.
* Add support for "pid=%p,user=%u,db=%d,app=%a,client=%h " and
  "user=%u,db=%d,app=%a,client=%h " log_line_prefix settings
  - This prefix misses a timestamp, but is useful when sending data over syslog.
* Log parsing: Correctly handle %a containing commas/square brackets
  - Note that this does not support all cases since Go's regexp engine
    does not support negative lookahead, so we can't handle an application
    name containing a comma if the log_line_prefix has a comma following %a.
* Ignore CSV log files in log directory [#83](https://github.com/pganalyze/collector/issues/83)
  - Some Postgres installations are configured to log both standard-format
    log files and CSV log files to the same directory, but the collector
    currently reads all files specified in a db_log_location, which works
    poorly with this setup.
* Tweak collector sample config file to match setup instructions
* Improvements to "--discover-log-location"
  - Don't keep running if there's a config error
  - Drop the log_directory helper command and just fetch the setting from Postgres
  - Warn and only show relative location if log_directory is inside
    the data directory (this requires special setup steps to resolve)
* Improvements to "--test-logs"
  - Run privilege drop test when running log test as root, to allow running
    "--test-logs" for a complete log setup test, avoiding the need to run
    a full "--test"
* Update pg_query_go to incorporate memory leak fixes
* Check whether pg_stat_statements exists in a different schema, and give a
  clear error message
* Drop support for Postgres 9.2
  - Postgres 9.2 has been EOL for almost 4 years
* Update to Go 1.16
  * This introduces a change to Go's certificate handling, which may break
    certain older versions of Amazon RDS certificates, as they do not
    include a SAN. When this is the case you will see an error message like
    "x509: certificate relies on legacy Common Name field".
  * As a temporary workaround you can run the collector with the
    GODEBUG=x509ignoreCN=0 environment setting, which ignores these incorrect
    fields in these certificates. For a permanent fix, you need to update
    your RDS certificates to include the correct SAN field: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.SSL-certificate-rotation.html


## 0.36.0      2021-01-21

* Config parsing improvements:
  * Fail fast when pganalyze section is missing in config file
  * Ignore duplicates in db_name config setting
    * Previously this could cause malformed snapshots that would be submitted
      correctly but could not be processed
  * Validate db_url parsing to avoid collector crash with invalid URLs
* Include pganalyze-collector-setup program (see 0.35 release notes) in supported packages
* Rename `<unidentified queryid>` query text placeholder to `<query text unavailable>`
  * This makes it clearer what the underlying issue is
* Revert to using `<truncated query>` instead of `<unparsable query>` in some situations
  * When a query is cut off due to pg_stat_activity limit being reached,
    show `<truncated query>`, to make it clear that increasing track_activity_query_size
    would solve the issue
* Ignore I/O stats for AWS Aurora utility statements
  * AWS Aurora appears to report incorrect blk_read_time and blk_write_time values
    for utility statements (i.e., non-SELECT/INSERT/UPDATE/DELETE); we zero these out for now
* Fix log-based EXPLAIN bug where query samples could be dropped if EXPLAIN failed
* Add U140 log event (inconsistent range bounds)
  * e.g.: ERROR:  range lower bound must be less than or equal to range upper bound
* Fix issue where incomplete schema information in snapshots was not marked correctly
  * This could lead to schema objects disappearing and being re-created
* Fix trailing newline handling for GCP and self-hosted log streams
  * This could lead to queries being poorly formatted in the UI, or some queries
    with single-line comments being ignored
* Include additional collector configuration settings in snapshot metadata for diagnostics
* Ignore "insufficient privilege" queries w/o queryid
  * Previously, these could all be aggregated together yielding misleading stats


## 0.35.0      2020-12-05

* Add new "pganalyze-collector-setup" program that streamlines collector installation
  * This is initially targeted for self-managed servers to make it easier to set up
    the collector and required configuration settings for a locally running Postgres
    server
  * To start, this supports the following environments:
    * Postgres 10 and newer, running on the same server as the collector
    * Ubuntu 14.04 and newer
    * Debian 10 and newer
* Collector test: Show server URLs to make it easier to access the servers in
  pganalyze after the test
* Collector test+reload: In case of errors, return exit code 1
* Ignore manual vacuums if the collector can't access pg_stat_progress_vacuum
* Don't run log test for Heroku, instead provide info message
  * Also fixes "Unsupported log_line_prefix setting: ' sql_error_code = %e '"
    error on Heroku Postgres
* Add pganalyze system user to adm group in Debian/Ubuntu packages
  * This gives the collector permission to read Postgres log files in a default
    install, simplifying Log Insights setup
* Handle NULL parameters for query samples correctly
* Add a skip_if_replica / SKIP_IF_REPLICA option (#117)
  * You can use this to configure the collector in a no-op mode on
    replicas (we only query if the monitored database is a replica), and
    automatically switch to active monitoring when the database is no
    longer a replica.
* Stop building packages for CentOS 6 and Ubuntu 14.04 (Trusty)
  * Both of these systems are now end of life, and the remaining survivor
    of the CentOS 6 line (Amazon Linux 1) will be EOL on December 31st 2020.


## 0.34.0      2020-11-07

* Check and report problematic log collection settings
  - Some Postgres settings almost always cause a drastic increase in log
    volume for little actual benefit. They tend to cause operational problems
    for the collector (due to the load of additional log parsing) and the
    pganalyze service itself (or indeed, likely for any service that would
    process collector snapshots), and do not add any meaningful insights.
    Furthermore, we found that these settings are often turned on
    accidentally.
  - To avoid these issues, add some client-side checks in the collector to
    disable log processing if any of the problematic settings are on.
  - The settings in question are:
     * [log_min_duration_statement](https://www.postgresql.org/docs/current/runtime-config-logging.html#GUC-LOG-MIN-DURATION-STATEMENT) less than 10ms
     * [log_statement](https://www.postgresql.org/docs/current/runtime-config-logging.html#GUC-LOG-STATEMENT) set to 'all'
     * [log_duration](https://www.postgresql.org/docs/current/runtime-config-logging.html#GUC-LOG-DURATION) set to 'on'
     * [log_error_verbosity](https://www.postgresql.org/docs/current/runtime-config-logging.html#GUC-LOG-ERROR-VERBOSITY) set to 'verbose'
  - If any of these are set to these unsupported values, all log collection will be
    disabled for that server. The settings are re-checked every full snapshot, and can be
    explicitly re-checked with a collector reload.
* Log Insights improvements
  * Self-managed server: Process logs every 3 seconds, instead of on-demand
  * Self-managed server: Improve handling of multi-line log events
  * Google Cloud SQL: Always acknowledge Pub Sub messages, even if collector doesn't handle them
  * Optimize stitching logic for reduced CPU consumption
  * Explicitly close temporary files to avoid running out of file descriptors
* Multiple changes to improve debugging in support situations
  * Report collector config in full snapshot
    - This reports certain collector config settings (except for passwords/keys/credentials)
      to the pganalyze servers to help with debugging.
  * Print collector version at beginning of test for better support handling
  * Print collection status and Postgres version before submitting snapshots
  * Change panic stack trace logging from Verbose to Warning
* Add support for running the collector on ARM systems
  * Note that we don't provide packages yet, but with this the collector
    can be built on ARM systems without any additional patches.
* Introduce API system scope fallback
  - This fallback is intended to allow changing the API scope, either based
    on user configuration (e.g. moving the collector between different
    cloud provider accounts), or because of changes in the collector identify
    system logic.
  - The new "api_system_scope_fallback" / PGA_API_SYSTEM_SCOPE_FALLBACK config
    variable is intended to be set to the old value of the scope. When the
    pganalyze backend receives a snapshot with a fallback scope set, and there
    is no server created with the regular scope, it will first search the
    servers with the fallback scope. If found, that server's scope will be
    updated to the (new) regular scope. If not found, a new server will be
    created with the regular scope. The main goal of the fallback scope is to
    avoid creating a duplicate server when changing the scope value
* Use new fallback scope mechanism to change scope for RDS databases
  - Previously we identified RDS databases by there ID and region only, but
    the ID does not have to be unique within a region, it only has to be
    unique within the same AWS account in that region. Thus, adjust the
    scope to include both the region and AWS Account ID (if configured or
    auto-detected), and use the fallback scope mechanism to migrate existing
    servers.
* Add support for GKE workload identity [Yash Bhutwala](https://github.com/yashbhutwala) [#91](https://github.com/pganalyze/collector/pull/91)
* Add support for assuming AWS instance roles
  - Set the role to be assumed using the new `aws_assume_role` / `AWS_ASSUME_ROLE`
    configuration setting. This is useful when the collector runs in a different
    AWS account than your database.



## 0.33.1      2020-09-11

* Ignore internal admin databases for GCP and Azure
  - This avoids collecting data from these internal databases, which produces
    unnecessary errors when using the all databases setting.
* Add log_line_prefix check to GCP self-test
* Schema stats handling: Avoid crash due to nil pointer dereference
* Add support for "%m [%p]: [%l-1] db=%d,user=%u " log_line_prefix


## 0.33.0      2020-09-03

* Add helper for log-based EXPLAIN access and use if available
  - This lets us avoid granting the pganalyze user any access to the data to follow
    the principle of least privilege
  - See https://pganalyze.com/docs/explain/setup/log_explain
* Avoid corrupted snapshots when OIDs get reused across databases
  - This would have shown as data not being visible in pganalyze,
    particularly for servers with many databases where tables were
    dropped and recreated often
* Locked relations: Ignore table statistics, handle other exclusive locks
   - Tables being rewritten would cause the relation statistics query to
      fail due to statement timeout (caused by lock being held)
   - Non-relation locks held in AccessExclusiveLock mode would cause all
      relation information to disappear, but only for everything thats not
      the top-level relation information. This is due to the behaviour of
      NOT IN when the list contains NULLs (never being true, even if an
      item doesn't match the list). The top-level relation information
      was using a LEFT JOIN that doesn't suffer from this problem. This likely
      caused problems reported as missing index information, or indices
      showing as being recently created even though they've exited for a
      while.
* Improvements to table partitioning reporting
* Enable additional settings to work correctly when used in Heroku/Docker
  - DB_NAME
  - DB_SSLROOTCERT_CONTENTS
  - DB_SSLCERT_CONTENTS
  - DB_SSLKEY_CONTENTS


## 0.32.0      2020-08-16

* Add `ignore_schema_regexp` / `IGNORE_SCHEMA_REGEXP` configuration option
  - This is like ignore_table_pattern, but pushed down into the actual
    stats-gathering queries to improve performance. This should work much
    better on very large schemas
  - We use a regular expression instead of the current glob-like matching
    since the former is natively supported in Postgres
  - We now warn on usage of the deprecated `ignore_table_pattern` field
* Add warning for too many tables being collected (and recommend `ignore_schema_regexp`)
* Allow keeping of unparsable query texts by setting `filter_query_text: none`
  - By default we replace everything with `<unparsable query>` (renamed
    from the previous `<truncated query>` for clarity), to avoid leaking
    sensitive data that may be contained in query texts that couldn't be
    parsed and that Postgres itself doesn't mask correctly (e.g. utility
    statements)
  - However in some situations it may be desirable to have the original
    query texts instead, e.g. when the collector parser is outdated
    (right now the parser is Postgres version 10, and some newer Postgres 12
    query syntax fails to parse)
  - To support this use case, a new "filter_query_text" / FILTER_QUERY_TEXT
    option is introduced which can be set to "none" to keep all query texts.
* EXPLAIN plans / Query samples: Support log line prefix without %d and %u
  - Whilst not recommended, in some scenarios changing the log_line_prefix
    is difficult, and we want to make it easy to get EXPLAIN data even in
    those scenarios
  - In case the log_line_prefix is missing the database (%d) and/or the user
    (%u), we simply use the user and database of the collector connection
* Log EXPLAIN: Run on all monitored databases, not just the primary database
* Add support for stored procedures (new with Postgres 11)
* Handle Postgres error checks using Go 1.13 error helpers
  - This is more correct going forward, and adds a required type check for
    the error type, since the database methods can also return net.OpError
  - Fixes "panic: interface conversion: error is *net.OpError, not *pq.Error"
* Collect information on table partitions
  - Relation parents as well as partition boundary (if any)
  - Partitionining strategy in use
  - List of partitioning fields and/or expression
* Log Insights: Track TLS protocol version as a log line detail
  - This allows verification of which TLS versions were used to connect to the
    database over time
* Log Insights: Track host as detail for connection received event
  - This allows more detailed analysis of which IPs/hostnames have connected
    to the database over time
* Example collector config: Use collect all databases option in the example
  - This improves the chance that this is set up correctly from the
    beginning, without requiring a backwards incompatible change in the
    collector


## 0.31.0      2020-06-23

* Add Log Insights support for Azure Database for PostgreSQL
* Log Insights: Avoid unnecessary "Timeout" error when there are other failures
* Log EXPLAIN: Don't run EXPLAIN logic when there are no query sample
* Improve non-fatal error messages to clarify the collector still works
* Log grant failure: Explain root cause better (plan doesn't support it / fair use limit reached)


## 0.30.0      2020-06-12

* Track local replication lag in bytes
* RDS: Handle end of log files correctly
* High-frequency query collection: Avoid race condition, run in parallel
  * This also resolves a memory leak in the collector that was causing
    increased memory usage over time for systems that have a lot of
    pg_stat_statements query texts (causing the full snapshot to take
    more than a minute, which triggered the race condition)


## 0.29.0      2020-06-02

* Package builds: Use Golang 1.14.3 patch release
  * This fixes https://github.com/golang/go/issues/37436 which was causing
    "mlock of signal stack failed: 12" on Ubuntu systems
* Switch to simpler tail library to fix edge case bugs for self-managed systems
  * The hpcloud library has been unmaintained for a while, and whilst
    the new choice doesn't have much activity, in tests it has shown
    to work better, as well as having significantly less lines of code
  * This also should make "--test" work reliably for self-managed systems
    (before this returned "Timeout" most of the time)
* Index statistics: Don't run relation_size on exclusively locked indices
  * Previously the collector was effectively hanging when it encountered an
    index that has an ExclusiveLock held (e.g. due to a REINDEX)
* Add another custom log line prefix: "%m %r %u %a [%c] [%p] "
* RDS fixes
  * Fix handling of auto-detection of AWS regions outside of us-east-1
  * Remember log marker from previous runs, to avoid duplicate log lines
* Add support for Postgres 13
  * This adds support for running against Postgres 13, which otherwise breaks
    due to backwards-incompatible changes in pg_stat_statements
  * Note that there are many other new statistics views and metrics that
    will be added separately


## 0.28.0      2020-05-19

* Add "db_sslkey" and "db_sslcert" options to use SSL client certificates
* Add Ubuntu 20.04 packages
* Update to Go 1.14, latest libpq
* Ensure that we set system type correctly for Heroku full snapshots
* Detect cloud providers based on hostnames from DB_URL / db_url as well
  * Previously this was only detected for the DB_HOST / db_host setting, and that is unnecessarily restrictive
  * Note that this means your instance may show up under a new ID in pganalyze after upgrading to this version
* Log Explain
  * Ignore pg_start_backup queries
  * Support EXPLAIN for queries with parameters
* Log Insights improvements
  * Experimental: Google Cloud SQL log download
  * Remove unnecessary increment of log line byte end position
  * Make stream-based log processing more robust
* Add direct "http_proxy" & similar collector settings for Proxy config
  * This avoids problems in some environments where its not clear whether
    the environment variables are set. The environment variables HTTP_PROXY,
    http_proxy, HTTPS_PROXY, https_proxy, NO_PROXY and no_proxy continue to
    function as expected.
* Fix bug in handling of state mutex in activity snapshots
  * This may have been the cause of "unlock of unlocked mutex" errors
    when having multiple servers configured.


## 0.27.0      2020-01-06

* Activity snapshot: Track timestamp of previous activity snapshot
* Support setting custom AWS endpoints using environment variables
* Increase allowed characters for pid field to 7 (for 64-bit systems)
  * This supports environments where pid_max is set to 4194304 instead of 32768
  * Note that this means continuity with old backend/vacuum identities is lost
    i.e. data might show up incorrectly for existing processes after the upgrade


## 0.26.0      2019-12-31

* Add new wait events from Postgres 12
* Avoid unnecessary allocations in ReplaceSecrets function
* Rename "aws_endpoint_rds_signing_region" to "aws_endpoint_signing_region"
  * This more accurately reflects how the setting is used. Backwards
    compatibility is provided, but its recommended to migrate to the new
    config setting (when in use for custom AWS API endpoints)


## 0.25.1      2019-12-18

* Update rds-ca-2019-root.pem file to be correct certificate
  * This was identical to the 2015 certificate by accident, causing
    connection errors


## 0.25.0      2019-12-18

* Add new "log_explain" mode, as an alternative to auto_explain (experimental)
  * Enable by setting "enable_log_explain: 1" or "PGA_ENABLE_LOG_EXPLAIN=1"
  * This is intended for providers such as Heroku Postgres where you can't use
    the auto_explain extension to send EXPLAIN plans into pganalyze


## 0.24.0      2019-12-05

* Add support for Azure Database for PostgreSQL
* Add support for Google Cloud SQL
* Use pg_has_role to determine pg_monitor membership
  * This improves handling of nested memberships, which previously were not
    detected correctly
* Generalize almost-superuser detection to support Azure and Cloud SQL better
* Add support for running "--test --reload" to test and reload if successful
  * This makes it easier to not forget reloading the collector after making
    a change


## 0.23.0      2019-11-27

* Vacuum progress: Ignore "(to prevent wraparound)" in query text
* Update distributions for packaging to reflect current versions
  * Remove Ubuntu Precise (its been EOL for 2 years)
  * Remove Fedora 24 (its been EOL for a while)
  * Add Fedora 29
  * Add Fedora 30
  * Add RHEL8
  * Add Debian 10 ("Buster")
* Import RDS 2019 CA root certificate
  * This is now available by specifying "db_ssl_root_cert = rds-ca-2019-root"
* Update builds and tests to use Go 1.13


## 0.22.0      2019-08-04

* Allow HTTP-only proxy connections when specified by the user
* Allow all replication LSN fields to be null
* Add full context for pg_stat_statements error messages
* Docker: Use entrypoint, provide easy "test" command, hide timestamps
* Amazon RDS
  * Add support for custom AWS endpoints
  * Include RDS root certificate for Docker builds
  * Automatically detect RDS instance ID from Docker env variables as well
  * Allow the ECS task metadata service
  * Show verbose AWS credentials chain errors


## 0.21.0      2019-05-21

* Self-hosted: Ignore additional file system types that are not important
* Fix helper process when using systemd service
  - This unfortunately requires us to remove the "NoNewPrivileges" mode, since
    we intentionally use a setuid binary (pganalyze-collector-helper) to be
    able to discover the data directory as well as determine the size
    of the WAL directory
* Increase systemd service memory limit to 1GB
  - We've previously limited this to 256MB for all use cases, which is too
    small when monitoring multiple systems
* Security: Lock down permissions for /etc/pganalyze-collector.conf
  - Previously this was world-readable, which may make credentials accessible
    to more system users than intended
  - Upgrading the packages will also apply this fix retroactively


## 0.20.0      2019-05-19

* Allow full snapshots to run when pg_stat_statements is not fully enabled
  - This provides for a smoother onboarding experience, as you can use the
    collector even if pg_stat_statements is not (yet) enabled through
    shared_preload_libraries
* RDS integration improvements
  - Don't attempt to connect to rdsadmin database
  - Accept rds_superuser as superuser for monitoring purposes
  - Correctly handle enhanced monitoring disk partitions
* Self-hosted system helper: Explicitly look for process called "postgres"
  - This avoids issues where there is another process that starts postgres
    and runs earlier than the actual postgres process


## 0.19.1      2019-04-12

* Add support for compact snapshots saving to local files


## 0.19.0      2019-04-11

* Enable logs and activity snapshots by default
  * Note that they are disabled when requested by server, to avoid overwhelming
    the server with compact snapshot grants
* Reduce memory consumption by only storing the required query texts
  * This also introduces two additional special query texts that get sent:
    - "\<pganalyze-collector\>" which identifies internal collector queries
    - "\<insufficient privilege\>" which identifies permission errors
* The collector now always normalizes query texts directly after retrieval
* Only display "You are not connecting as superuser" message during tests


## 0.18.1      2019-03-13

* Config: Fix crash when configuration is read from environment only


## 0.18.0      2019-03-05

* Add Postgres 12 support
* Process each configured server section in parallel
  - This avoids problems when a high number of servers is configured, since
    previously they would be processed serially, leading to skewed statistics
    for servers processed later in the sequence
* Introduce log filtering for PII and other kinds of secrets
  - This is controlled by the new "filter_log_secret" configuration setting
* Remove explicit connection to EC2 metadata service [#27](https://github.com/pganalyze/collector/issues/27)
* Gather total partitioned/inheritance children table size
* Correctly retrieve distributed table size for Citus extension tables
* Ensure connections are encrypted and made using TLS 1.2
* Build improvements
  - Update builds and tests to use Go 1.12
  - Switch to new Go module system instead of gvt


## 0.17.1      2018-12-31

* Vacuum monitoring
  - Filter out results with insufficient privileges
    - Previously we would error out hard in this case, which isn't helpful and
      can stop the usage of activity snapshots on shared systems
  - Correctly close DB connection on error
* Connection establishment: Make sure to close connection on early errors
* Add support for "%t [%p]: [%l-1] [trx_id=%x] user=%u,db=%d " log_line_prefix
* Add LOG_LOCATION environment config variable (same as db_log_location)
  * Note: We don't support the equivalent of the experimental setting
    db_log_docker_tail since it would require the "docker" binary inside
    the pganalyze container (as well as full Docker access), instead the
    approach for using pganalyze as a sidecar container alongside Postgres
    currently requires writing to a file and then mounting that as a
    volume inside the pganalyze container


## 0.17.0      2018-11-26

* TOAST handling
  - Track size of TOAST table separately
    - This can often be useful to determine whether the bulk of a table is in
      TOAST storage, or in the main storage, and thus reads may behave slightly
      differently
  - Fix bug in detection of HasToast for tables
    - Previously we recorded this the wrong way around, i.e. tables that had
      no TOAST would have been flagged as having TOAST. This hasn't been used
      thusfar in terms of stats processing, but might be in the future, so
      better to have this correct
  - Track TOAST flag for autovacuum and buffercache statistics
* Schema-qualify functions/tables wherever possible
  - Whilst not much of problem in practice, since the collector doesn't run
    as superuser, it doesn't hurt to schema-qualify everything
  - This also introduces an explicit "SCHEMA public" for CREATE EXTENSION
    statements to support non-standard search paths better
* Extract schema/relation name from autovacuum log events
  - This is done to make it easier to link autovacuum log events to the
    corresponding vacuum statistics records
* Include partitioned base tables in the table information gathered
* Historic statement stats: Ignore any data older than 1 hour
  - There has been some cases where the state structure doesn't get reset
    and the historic statement stats keeps growing and growing. Add this
    as a safety measure to ensure a run can complete successfully


## 0.16.0      2018-10-16

* Fix scoping of on-disk state to reflect system type/scope/id
  * Previously we only considered API key to determine which state to save,
    which meant that in configurations with multiple servers but a single
    organization API key we'd usually loose statistics on restarts, or
    get the wrong values for diff-ing the query statistics
  * This is a backwards-incompatible change for the on-disk format and
    therefore bumps the version from 1 to 2. Effectively this will show
    as one period of no data after upgrading to this version, as the
    previously saved counter values in the state file won't be used
* Add support for "%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h " log_line_prefix
* Use explicit log file clean up instead of deferrals
  * There have been reports of old temporary files containing log lines
    not being cleaned up fully - attempt to fix that
* systemd config: Fix incorrect specification of memory limit & restart event [Dom Hutton](https://github.com/Dombo) [#26](https://github.com/pganalyze/collector/pull/26)


## 0.15.2      2018-09-28

* Fix supported log_line_prefix list to include recently added prefixes


## 0.15.1      2018-09-28

* Add additional supported log_line_prefix settings
  * '%t [%p]: [%l-1] user=%u,db=%d - PG-%e '
* Add new log classifications
  * WAL_BASE_BACKUP_COMPLETE
  * SERVER_STATS_COLLECTOR_TIMEOUT
* Add "sorry, too many clients already" out of connections log classification


## 0.15.0      2018-09-27

* Add additional supported log_line_prefix settings
  * 1) '%m [%p] %q[user=%u,db=%d,app=%a] '
  * 2) '%m [%p] %q[user=%u,db=%d,app=%a,host=%h] '
* Add support for [local] when using %r in log_line_prefix
* Make --discover-log-location work when using monitoring user
* Correctly detect pg_wal directory for Postgres 10 and newer
* Introduce setting for maximum collector connections
  * This previously existed as a hard-coded 5 connection maximum based on the
    pganalyze-collector application name in pg_stat_activity
  * Adds "max_collector_connections" configuration setting to override
  * Increases default max connections to 10 to better support activity snapshots
  * Writes an error to the log instead of panicing when limit is reached
* Add experimental support for Docker log monitoring
  * Adds "db_log_docker_tail" setting to specify the container name
  * Allows monitoring the logs of a Postgres instance running inside
    Docker, when running the collector outside (on the Docker Host)


## 0.14.4      2018-08-23

* Add support for Postgres 11
* Support log_hostname=1 when using log_line_prefix that contains %r
* Duplicate config detection: Differentiate by API key and API base url


## 0.14.3      2018-08-08

* Add configuration setting to disable high-frequency query statistics


## 0.14.2      2018-07-21

* Log parser: Add support for offset-based timezone identifier
  - Previously we assumed that Postgres always outputs the timezone name,
    that is not correct for some timezones, where instead an offset like
    "+0100" would be printed.


## 0.14.1      2018-07-18

* Fixes to experimental sequences report
* Document monitoring helper functions for sequences report


## 0.14.0      2018-07-15

* Introduce once-per-minute query statistics collection
  - Enabled for PostgreSQL 9.4+
  - This replaces the old logic for query stats without text (statement_text_frequency),
    and is always active
  - Statistics data gets sent with every full snapshot
* Backend counts: Support retrieving stats for Postgres 9.5 and older
* Log Insights: Add support for detecting aggressive vacuums (Postgres 11+)
* Parse serialization failure log events (U138 and U139)
* Have systemd restart the collector after crashes [Dom Hutton](https://github.com/Dombo) [#23](https://github.com/pganalyze/collector/pull/23)


## 0.13.1      2018-06-18

* Don't error out on pg_stat_replication.replay_lsn being NULL [#21](https://github.com/pganalyze/collector/issues/21)


## 0.13.0      2018-06-15

* Support basic vacuum information for Postgres 9.5 and older
* Track connection counts per database and per role
* Add ignore_table_pattern / IGNORE_TABLE_PATTERN option
* Avoid errors when collecting from AWS Aurora for Postgres instances
* Log Insights improvements
  * Better setup help
  * Increase read rate for RDS log downloader from 100 to 1000 lines
  * Add support for extracting EXPLAIN plans from auto_explain output
  * Fix autovacuum, autoanalyze and checkpoint completed parsing for PG 10
  * Avoid sending unparsed explain text for truncated log lines
  * Detect vacuum/analyze skipped log lines


## 0.12.0      2018-03-05

* Set username/database name correct for Heroku logstreams
* Support additional ryslog format that contains user/db/app information
* Update to Go 1.10


## 0.11.0      2018-01-31

* Add support for Log Insights on self-hosted systems
* Add additional log classifications, speed up analysis by reusing regexps
* Add "--analyze-logfile" option to test collector with local logfiles
* Associate truncated queries to the correct error fingerprint
* Update to Go 1.9.3


## 0.10.0      2017-10-31

* Update pg_query_go to Postgres 10 and fingerprint version 2
  - This is a breaking change in collector output, as queries will now
    be fingerprinted differently
* Activity snapshots
  - Use pg_stat_activity helper when it exists
  - Track VACUUM progress in activity snapshots
  - Activity data: Ignore backends that are not visible to the user
  - Allow additional digits for PID in pg_stat_activity [Joseph Bylund](https://github.com/jbylund)
* Don't collect backend data for full snapshot anymore, this is all delegated
  to activity snapshots now
* Update to Go 1.9.2
* RDS pgss check: Add additional safety against nil pointer dereferences


## 0.9.17      2017-10-09

* Logs: Fix regexp for 9.5 vacuum output (skip pins, but not skip frozen)
* Update to Go 1.9.1 release
* Allow disabling collection of relation/schema information
* Add experimental activity snapshots
  - This is not for public consumption yet, and trying to use it will result in
    an error from the server - but watch this space :)


## 0.9.16      2017-10-05

* Support for Postgres 10 monitoring role
* Log Insights improvements
  - Fix bug where referenced query wouldn't be correctly identified
  - Collect query text and parameters for all query samples
  - Fix issues with Heroku Postgres log collection


## 0.9.15      2017-10-01

* Update pg_query_go / libpg_query to 9.5-1.6.2
  * Updates the query fingerprinting logic to avoid seeing different
    FETCH/DECLARE/CLOSE cursor names as unique queries - statistics on this
    are not going to be useful in most cases, and will clog the processing
    pipeline
  * Updates the query fingerprinting logic to ignore the table name for
    CREATE TEMPORARY TABLE
  * Updates the query fingerprinting logic to better handle the values list
    for INSERT statements to group complex, but similar statements together
* Support specifying db_sslmode=verify-full and passing certificate information
  using db_sslrootcert / db_sslrootcert_contents
  * The collector packages now also ship a set of known DB-as-a-Service CA
    certificates, starting with the often needed rds-ca-2015-root certificate
    (just pass that term instead of a path to db_sslrootcert)
* Support for Postgres 10
* Heroku: Support specifying configuration name in log drain endpoints
* RDS: Ensure to delete temporary log files quickly after they are submitted


## 0.9.14      2017-06-06

* Add support for Heroku logdrains


## 0.9.13      2017-05-17

* Log Monitoring
  - Upload encrypted log data to S3, and only send byte ranges in snapshot
  - Implement log classification
* Add --version flag to show current collector version
* Replication stats: Allow replay location to be null
* Add support for error and success callbacks
* Introduce server-controlled ability to reset pg_stat_statements


## 0.9.12      2017-04-05

* SystemScope: Include DbAllNames status for local collections
* Fix wording of some log messages
* Refactor log collection and query sampling / explaining
* Introduce ability to collect statement text less often
* Make statement timeout a server-controlled option
* Allow enabling/disabling automatic EXPLAIN from server-side


## 0.9.11      2017-03-01

* Fix collection of replication statistics for non-superusers
* Add monitoring helper for replication statistics


## 0.9.10      2017-02-27

* Update to Go 1.8 in all builds
* Disable verbose logging on Heroku
* Add SystemID for all types of systems
* Change default config to be in account-based format
* Support collecting schema info/stats from multiple databases per server
* Allow monitoring all databases using DB_ALL_NAMES=1 env variable
* Fix issue with helper having wrong executable format
* Cleanup test mechanism in test/ folder
* Handle null relation sizes for temp tables
* Collect replication statistics


## 0.9.9       2016-12-29

* Fix edge case that made RDS system metrics code crash
* Add VACUUM and Sequence reports


## 0.9.8       2016-12-19

* Make bloat report work under the restricted user
* Add option to run a Go performance trace on a single test run
* Improve error tracking
* Update pg_query_go
  * Cut off fingerprints at 100 nodes deep to avoid excessive runtimes/memory


## 0.9.7       2016-11-02

* Prevent leaks of previous scheduler runs when reloading.


## 0.9.6       2016-11-01

* New Heroku support based on user API keys
* Support for new Reports feature (in private beta right now)
* Add PGA_ALWAYS_COLLECT_SYSTEM_DATA to force collection of system data
* Increase statement timeout to 30 seconds to account for some larger databases
* Support for writing snapshots to local filesystem (needed by pganalyze Enterprise)


## 0.9.5       2016-09-21

* Improved first user experience
  * Add "--reload" command for sending SIGHUP to daemon process
  * Show error message when configuration file is empty
* Experimental build support for Solaris
* System metrics: Various fixes
* Packaging: Add support for Ubuntu Precise / 12.04 LTS


## 0.9.4       2016-08-16

* Introduce "pganalyze-collector-helper": Setuid Binary that can be used to run
  privileged actions when the main collector is running as non-root (the default)
* Determine the correct distance between two collector runs (instead of assuming 600 seconds)
* Better monitoring for self-hosted systems
  * Collect missing Disk I/O statistics
  * Fix calculation logic for disk utilization
  * Collect kernel version and architecture
  * Don't monitor the local loopback network interface
  * Sort disk/partition/network interface names before output
  * Don't collect local system information when monitoring remote hosts
* Packaging
  * Update to Go 1.7
  * Systemd: Enforce memory limit of 256mb for the collector


## 0.9.3       2016-08-07

* Correctly identify PostgreSQL data directory and pg_xlog location
* Avoid potential NaN values in disk stats for self-hosted systems
* Don't write state file for dry runs by default


## 0.9.2       2016-08-01

* PostgreSQL 9.2, 9.3 and 9.6 Support
* Adjust default config and state file path to match packages
* Allow using postgres driver default values for connection credentials


## 0.9.1       2016-07-28

* Add support for logging to syslog instead of stderr
* Init scripts for systemd, upstart and sysvinit (see contrib/ directory)
* Packaging scripts for common Linux distributions (see packages/ directory)


## 0.9.0       2016-07-14

* First official release of new protocol buffers-based collector


## 0.9.0rc8    2016-07-08

* Significant restructuring of the codebase
  * We're now sending data using the protocol buffers format
  * Snapshot data is directly uploaded to S3
* Query, table and system statistics are diff-ed on the client side
* Support for monitoring system metrics on self-hosted systems is added again
* New safety mechanisms against stuck/slow collector runs


## 0.9.0rc7    2016-04-14

* Add support for RDS enhanced monitoring
* Simplify dependencies and document OSS licenses in use


## 0.9.0rc6    2016-04-07

* Bugfixes for AWS Instance Role handling


## 0.9.0rc5    2016-04-07

* Introduce new --diff-statements option (default off for now)
  * This calculates the diff for the counter values of pg_stat_statements on the client (i.e. collector),
    instead of the server for increased accuracy and protection against out-of-order processing
* Introduce "opts" to the snapshot, for indicating which options were chosen
* Never open more than 1 connection to the same database
  * This covers edge cases like sending a lot of SIGHUP signals
  * In case we detect more than 1 connection we error out and exit, to avoid
    clogging the database
* Use AWS EC2 instance role if no credentials are specified


## 0.9.0rc4    2016-04-03

* Ensure pg_toast schema is excluded when calculating index bloat


## 0.9.0rc3    2016-03-28

* Send Postgres version to the server as well


## 0.9.0rc2    2016-03-27

* Add --test mode to ease initial setup
* Don't do an initial run when daemonizing (the default), this is mostly so we
  can keep a clean schedule and prevent issues if the config is temporarily wrong,
  or the server is unreachable for some reason
* Use POSIX commandline flags (double dash instead of single dash, shorthand flags)
  instead of Go's flag approach
* Added support for specifying sslmode when connecting, and default to "prefer"
   * This also fixes an issue where beforehand we required SSL to always be present
   * For maximum security you might want to set this to "require" or "verify-full"
* Re-introduced all required statistics currently used by pganalyze


## 0.9.0rc1    2016-03-22

* Initial Go re-release
* The collector now runs as a daemon (instead of through the crontab)
* We optionally write a pidfile, which you can use to SIGHUP for config changes
* You can specify multiple databases in the configuration file
* Support for fetching Amazon Web Service RDS data (CloudWatch and log files)


---

Changelog of original Python-based collector:

## 0.8.0    2015-04-08

* Compress data using zlib by default (disable with --no-compression)
* Collect function definitions (disable with --no-postgres-functions)
* Collect normal and materialized views
* Update table/index bloat queries to use newest by pgExperts/Josh Berkus
  * Disable bloat stats collection with --no-postgres-bloat
* Output timing information in verbose mode
* Added Dockerfile to enable running collector as a sidekick service


## 0.7.1    2015-03-16

* Improved monitoring user support
  * Add support for collecting backend info as restricted user
  * Gracefully fail if we are not superuser
* psycopg2: Fix bugs, set connection timeout (10s)
* pg8000: Default to SSL connections, fallback to non-SSL


## 0.7.0    2015-03-16

* Reset-less collector
  * Calculate the diff on the receiver end for simplicity's sake
  * Don't run pg_stat_statements_reset() anymore
  * Make --no-reset option a no-op (will be removed soon)
* Restricted privileges
  * Added support for using a monitoring user (see README)
  * Removed hard superuser requirement
* Don't collect query information from pg_stat_activity (it might be sensitive)
  * Remove --no-query-parameters option, its a no-op now
* Collect replication statistics
* Add option for disabling collection of postgres locks & config settings
* Remove support for pg_stat_plans, its not supported anymore
* Update vendored pg8000 to latest version (1.10.1)


## 0.6.4    2014-11-17

Fixes:

* Do not require dbhost to be set


## 0.6.3    2014-11-16

Re-release to fix build issues


## 0.6.2    2014-11-13

Fixes:

* CREATE EXTENSION IF NOT EXISTS pg_stat_statements
* Auto-detect Amazon Web Service DB hosts as remote


## 0.6.1    2014-07-26

Fixes:

* Drop dmidecode dependency to gather server vendor/model on Linux
* Fix whitespace in generated configuration file


## 0.6.0    2014-07-18

Features & improvements:

* support for pg8000 as psycopg2 replacement
* collector can now be run from a zip file
* Split collector into separate modules
* import pg8000 & colorama
* Add conversation module for setup wizard
* Support for packaging to deb & rpm using fpm
* Move zip building to Makefile

Fixes:

* Don't replace newlines in collected queries with whitespace
* Ignore queries belonging to other databases


## 0.5.0	2014-01-24

* pg_stat_statements support


## 0.4.0	2013-12-04

* Switch from psql wrapper to psycopg2


## 0.3.2	2013-08-13

* Collect CPU information from OS
* Ignore queries created by the collector


## 0.3.1	2013-07-23

* Collecting more Postgres Information
  * GUCs (configuration settings)
  * BGWriter
  * backends
  * locks


## 0.3.0	2013-04-03

* Collect information about Postgres schema
  * Tables
  * Indexes
  * Bloat


## 0.2.0	2013-02-22

* Switch to Python
* dry-run mode - see data before it's posted
* privacy mode - don't send query examples to API
* Collect OS (CPU, memory, storage) information


## 0.1.4	2013-02-06

* Small fixes to config parsers and plan fetching


## 0.1.2	2013-01-29

* dry-run mode


## 0.0.1	2012-12-22

* Initial release of the Ruby Collector
* Support for fetching information from pg_stat_plans
