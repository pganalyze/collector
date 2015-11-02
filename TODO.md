## Basic working version

* [X] Use pganalyze-collector JSON format
* [X] Run queries at 10 minute interval
* [ ] DB: Indices
* [ ] Dockerfile


## Solve existing pganalyze-collector use cases

* [ ] Generate configuration file
* [ ] Use statistics helper if it exists
* [ ] Create pg_stat_statements extension if it exists
* [ ] Use version detection to fetch correct fields
* [ ] Compress sent data using zlib
* [ ] System information
* [ ] DB: Bloat, bgwriter, replication, locks, functions, settings
* [ ] DB: View definitions
* [ ] DB: Constraints


## Proxy

* [ ] HTTP API


---

=> Shared library that can be used by collector, proxy and API (for interface definition)
