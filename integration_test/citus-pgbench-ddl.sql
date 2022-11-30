ALTER TABLE pgbench_accounts DROP CONSTRAINT pgbench_accounts_pkey;
ALTER TABLE pgbench_accounts ADD PRIMARY KEY (aid, bid);
ALTER TABLE pgbench_tellers DROP CONSTRAINT pgbench_tellers_pkey;
ALTER TABLE pgbench_tellers ADD PRIMARY KEY (tid, bid);
SELECT create_distributed_table('pgbench_accounts', 'bid');
SELECT create_distributed_table('pgbench_branches', 'bid');
SELECT create_distributed_table('pgbench_history', 'bid');
SELECT create_distributed_table('pgbench_tellers', 'bid');