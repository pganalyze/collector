## pganalyze Snapshots

Snapshots are the format in which the pganalyze collector and the statistics processor
communicate. Snapshots are encoded using Protocol Buffers.

The collector takes statistics data from a PostgreSQL database server, puts it into an
easy-to-consume format, and encodes it using Protocol Buffers.

See also the [collector](https://github.com/pganalyze/collector) output/ directory.

## Definitions

**Reference:**

Term(s) that can be UPSERTed into the monitoring system, and is
referenced by its list index within other parts of the snapshot.

The goal here is that we can do a two-step processing of the data,
first we create/find all references and get their IDs, then we
just COPY all statistics into the database, replacing idx for ID.

**Information:**

Data that is attached to a reference, at most once, and is not stored historically.
When processing this data can simply be UPDATEd after the initial UPSERT.

In some cases this data might also be provided at less frequent intervals than Statistics.

**Statistic:**

Data that is attached to a reference, at most once, and is stored historically.
When processing this data can be COPYed after the initial UPSERT.

In case the input data is a counter, this will be normalized by the collector
before the snapshot is created, the recipient does not need to look at previous
values to find out what happened.

**Event:**

Data that is attached to a reference, and can occur multiple times for one reference within the snapshot.
When processing this data can be COPYed after the initial UPSERT.

## LICENSE

Copyright (c) 2016 pganalyze<br>
Licensed under the MIT License.
