pganalyze collector
===================

This is a Go-based daemon which collects various information about Postgres databases
as well as queries run on it.

All data is converted to a protocol buffers structure which can then be used as data source for monitoring & graphing systems. Or just as reference on how to pull information out of PostgreSQL.

It currently collections information about

 * Schema
   * Tables (including column, constraint and trigger definitions)
   * Indexes
 * Statistics
   * Tables
   * Indexes
   * Database
   * Queries
 * OS
   * CPU
   * Memory
   * Storage

Installation
------------

The collector is available in multiple convenient options:

* APT/YUM packages: https://packages.pganalyze.com/
* Docker sidekick service, see details further down in this file

Configuration (APT/YUM Packages)
--------------------------------

After the package was installed, you can find the configuration in /etc/pganalyze-collector.conf

Adjust the values in that file by adding your API key (found in the pganalyze dashboard, use one per database server), and database connection credentials.

You can repeat the configuration block with a different `[name]` if you have multiple servers to monitor.

See https://pganalyze.com/docs for further details.


Setting up a Restricted Monitoring User
---------------------------------------

By default pg_stat_statements does not allow viewing queries run by other users,
unless you are a database superuser. Since you probably don't want monitoring
to run as a superuser, you can setup a separate monitoring user like this:

```
CREATE SCHEMA pganalyze;

CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

/* HINT: Postgres 9.3 does not know the showtext-flag in public.pg_stat_statements, so you need to replace with with: */
/* SELECT * FROM public.pg_stat_statements(); */
/* in the following block, if you use 9.3 (The showtext-flag has been added in 9.4) */

CREATE OR REPLACE FUNCTION pganalyze.get_stat_statements(showtext boolean = true) RETURNS SETOF pg_stat_statements AS
$$
  /* pganalyze-collector */ SELECT * FROM public.pg_stat_statements(showtext);
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;

CREATE OR REPLACE FUNCTION pganalyze.get_stat_activity() RETURNS SETOF pg_stat_activity AS
$$
  /* pganalyze-collector */ SELECT * FROM pg_catalog.pg_stat_activity;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;

CREATE OR REPLACE FUNCTION pganalyze.get_column_stats() RETURNS SETOF pg_stats AS
$$
  /* pganalyze-collector */ SELECT schemaname, tablename, attname, inherited, null_frac, avg_width,
  n_distinct, NULL::anyarray, most_common_freqs, NULL::anyarray, correlation, NULL::anyarray,
  most_common_elem_freqs, elem_count_histogram
  FROM pg_catalog.pg_stats;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;

CREATE OR REPLACE FUNCTION pganalyze.get_stat_replication() RETURNS SETOF pg_stat_replication AS
$$
  /* pganalyze-collector */ SELECT * FROM pg_catalog.pg_stat_replication;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;

CREATE USER pganalyze WITH PASSWORD 'mypassword' CONNECTION LIMIT 5;
REVOKE ALL ON SCHEMA public FROM pganalyze;
GRANT USAGE ON SCHEMA pganalyze TO pganalyze;
```

Note that these statements must be run as a superuser (to create the `SECURITY DEFINER` function),
but from here onwards you can use the `pganalyze` user instead.

The collector will automatically use the helper methods
if they exist in the `pganalyze` schema - otherwise data will be fetched directly.

If you are on Postgres 9.6 and use activity snapshots:

```
CREATE OR REPLACE FUNCTION pganalyze.get_stat_progress_vacuum() RETURNS SETOF pg_stat_progress_vacuum AS
$$
  /* pganalyze-collector */ SELECT * FROM pg_catalog.pg_stat_progress_vacuum;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;
```

If you are using the Buffer Cache report in pganalyze, you will also need to create this additional helper method:

```
CREATE EXTENSION IF NOT EXISTS pg_buffercache;
CREATE OR REPLACE FUNCTION pganalyze.get_buffercache() RETURNS SETOF public.pg_buffercache AS
$$
  /* pganalyze-collector */ SELECT * FROM public.pg_buffercache;
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;
```

If you enabled the optional reset mode (usually not required), you will also need this helper method:

```
CREATE OR REPLACE FUNCTION pganalyze.reset_stat_statements() RETURNS SETOF void AS
$$
  /* pganalyze-collector */ SELECT * FROM public.pg_stat_statements_reset();
$$ LANGUAGE sql VOLATILE SECURITY DEFINER;
```


Example output
--------------

To get a feel for the data that is collected you can have a look at the following example:

 * [JSON w/ prettyprinting](https://gist.github.com/terrorobe/7103234)


Docker Container (RDS)
----------------------

If you are monitoring an RDS database and want to run the collector inside Docker, we recommend the following:

```
docker pull quay.io/pganalyze/collector:stable
docker run --rm --name pganalyze-mydb -e DB_URL=postgres://username:password@hostname.us-east-1.rds.amazonaws.com/mydb \
-e PGA_API_KEY=YOUR_PGANALYZE_API_KEY -e AWS_INSTANCE_ID=my-instance-id -e AWS_REGION=us-east-1 quay.io/pganalyze/collector:stable
```

You'll need to set PGA_API_KEY, AWS_INSTANCE_ID and AWS_REGION with the correct values.

Please also note that the EC2 instance running your Docker setup needs to have an IAM role that allows Cloudwatch access: https://pganalyze.com/docs/install/amazon_rds/03_setup_iam_policy

To get better data quality for server metrics, enable "Enhanced Monitoring" in your RDS dashboard. The pganalyze collector will automatically pick this up and get all the metrics.

We currently require one Docker container per RDS instance monitored.


Docker Container (non-RDS)
--------------------------

If the database you want to monitor is running inside a Docker environment you can use the Docker image:

```
docker pull quay.io/pganalyze/collector:stable
docker run --name my-app-pga-collector --link my-app-db:db --env-file collector_config.env quay.io/pganalyze/collector:stable
```

collector_config.env needs to look like this:

```
PGA_API_KEY=$YOUR_API_KEY
PGA_ALWAYS_COLLECT_SYSTEM_DATA=1
DB_NAME=your_database_name
DB_USERNAME=your_database_user
DB_PASSWORD=your_database_password
```

The only required arguments are PGA_API_KEY (found in the [pganalyze](https://pganalyze.com/) dashboard) and DB_NAME. Only specify `PGA_ALWAYS_COLLECT_SYSTEM_DATA` if the database is running on the same host and you'd like the collector to gather system metrics (from inside the container).

Note: You can add ```-v /path/to/database/volume/on/host:/var/lib/postgresql/data``` in order to collect I/O statistics from your database (this requires that it runs on the same machine).


Heroku Monitoring
-----------------

When monitoring a Heroku Postgres database, it is recommended you deploy the collector as its own app inside your Heroku account.

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/pganalyze/collector)

Follow the instructions in the pganalyze documentation to add your databases to the collector.


Success/Error Callbacks
-----------------------

In case you want to run a script based on data collection running successfully
and/or failing, you can set the `success_callback` and `error_callback` options:

```
[pganalyze]
...
error_callback=/usr/local/bin/my_error_script.sh

[mydb]
...
```

Note that the callback is executed in a shell, so you can use shell expressions as well.

The script will also have the following environment variables set:

* PGA_CALLBACK_TYPE (type of callback, `error` or `success`)
* PGA_CONFIG_SECTION (server that was processed, `mydb` in this example)
* PGA_SNAPSHOT_TYPE (type of data that was processed, currently there are `full` snapshots, as well as `logs` snapshots which contain only log data)
* PGA_ERROR_MESSAGE (error message, in the case of the error callback)


Authors
-------

 * [Lukas Fittl](https://github.com/lfittl)
 * [Michael Renner](https://github.com/terrorobe)


License
-------

pganalyze-collector is licensed under the 3-clause BSD license, see LICENSE file for details.
