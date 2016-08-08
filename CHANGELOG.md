# Changelog

## 0.9.3       2016-08-07

* Correctly identify PostgreSQL data directory and pg_xlog location
* Avoid potential NaN values in disk stats for self-hosted systems
* Don't write stats update for dry runs by default


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
