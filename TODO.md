## Solve existing pganalyze-collector use cases

* [ ] Generate configuration file
* [ ] Use statistics helper if it exists
* [ ] Create pg_stat_statements extension if it exists
* [ ] Use version detection to fetch correct fields
* [X] Compress sent data using zlib
* [ ] System information
* [ ] Option to not collect view statistics
* [ ] DB: Bloat, bgwriter, replication, locks, functions, settings
* [ ] DB: View definitions
* [ ] DB: Constraints


## Neat things

* [ ] Include normalized query in backends information

## FUTURE

* Sends diffs (i.e. what actually happened) and time ranges, instead of counters + timestamps
* Only sends (full) schema information on changes
* Short interval monitoring for 9.4+ pg_stat_statements (e.g. every 10s)
