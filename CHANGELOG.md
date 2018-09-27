# Changelog

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
