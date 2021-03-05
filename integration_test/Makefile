DOCKER_BUILD_OPTS = -t pganalyze-collector-test ..
DOCKER_RUN_CMD = docker run --name pganalyze-collector-test \
  --mount type=bind,source=/sys/fs/cgroup,target=/sys/fs/cgroup \
  --mount type=bind,source=/sys/fs/fuse,target=/sys/fs/fuse \
  --mount type=tmpfs,destination=/tmp \
  --mount type=tmpfs,destination=/run \
  --mount type=tmpfs,destination=/run/lock \
  -v `pwd`/docker-entrypoint-initdb.d:/docker-entrypoint-initdb.d/ \
  -e POSTGRES_HOST_AUTH_METHOD=trust \
  -d \
  pganalyze-collector-test

.PHONY: pg93 pg94 pg95 pg96 pg10 pg11 pg12 pg13 guided-setup installer

all: pg93 pg94 pg95 pg96 pg10 pg11 pg12 pg13 guided-setup installer

pg93:
	docker build -f Dockerfile.test-pg93 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg93_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg93_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg93_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg93_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg93_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg93_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg93_2.json | sort --version-sort -f > pg93.out
	diff -Naur pg93.expected pg93.out && echo 'success'

pg94:
	docker build -f Dockerfile.test-pg94 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg94_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg94_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg94_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg94_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg94_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg94_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg94_2.json | sort --version-sort -f > pg94.out
	diff -Naur pg94.expected pg94.out && echo 'success'

pg95:
	docker build -f Dockerfile.test-pg95 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg95_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg95_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg95_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg95_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg95_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg95_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg95_2.json | sort --version-sort -f > pg95.out
	diff -Naur pg95.expected pg95.out && echo 'success'

pg96:
	docker build -f Dockerfile.test-pg96 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg96_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg96_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg96_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg96_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg96_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg96_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg96_2.json | sort --version-sort -f > pg96.out
	diff -Naur pg96.expected pg96.out && echo 'success'

pg10:
	docker build -f Dockerfile.test-pg10 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg10_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg10_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg10_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg10_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg10_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg10_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg10_2.json | sort --version-sort -f > pg10.out
	diff -Naur pg10.expected pg10.out && echo 'success'

pg11:
	docker build -f Dockerfile.test-pg11 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg11_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg11_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg11_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg11_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg11_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg11_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg11_2.json | sort --version-sort -f > pg11.out
	diff -Naur pg11.expected pg11.out && echo 'success'

pg12:
	docker build -f Dockerfile.test-pg12 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg12_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg12_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg12_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg12_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg12_2.json | grep --quiet pgbench_history; then \
		echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg12_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg12_2.json | sort --version-sort -f > pg12.out
	diff -Naur pg12.expected pg12.out && echo 'success'

pg13:
	docker build -f Dockerfile.test-pg13 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg13_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg13_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg13_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg13_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg13_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg13_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg13_2.json | sort --version-sort -f > pg13.out
	diff -Naur pg13.expected pg13.out && echo 'success'

guided-setup:
	docker build -f Dockerfile.test-guided-setup $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test systemctl start postgresql
	docker exec --user pganalyze --detach pganalyze-collector-test pganalyze-collector

	docker exec --privileged pganalyze-collector-test env PGA_SETUP_COLLECTOR_TEST_EXTRA_ARGS='--dry-run' pganalyze-collector-setup --api-key=abc123 --recommended --db-name=postgres

	docker exec --user postgres pganalyze-collector-test psql -c "SHOW shared_preload_libraries" \
		-c "SELECT count(*) FROM pg_settings WHERE pending_restart" \
		-c "SELECT name, setting FROM pg_settings WHERE name IN ('log_duration', 'log_error_verbosity', 'log_line_prefix', 'log_min_duration_statement', 'log_statement', 'auto_explain.log_analyze', 'auto_explain.log_buffers', 'auto_explain.log_timing', 'auto_explain.log_triggers', 'auto_explain.log_verbose', 'auto_explain.log_format', 'auto_explain.log_min_duration', 'auto_explain.log_nested_statements') ORDER BY name" \
		-c "SELECT pg_has_role('pganalyze', 'pg_monitor', 'usage')" \
		> guided-setup.postgres.out
	docker exec --user pganalyze pganalyze-collector-test cat /etc/pganalyze-collector.conf > guided-setup.collector-config.out
	docker exec pganalyze-collector-test sh -c "PGA_DISABLE_ACTIVITY=1 pganalyze-collector --dry-run --force-state-update -v" > guided-setup.snapshot.json.out 2>guided-setup-snapshot.log.out

	docker exec --user postgres pganalyze-collector-test pg_ctl --pgdata /var/lib/postgresql/10/main --wait stop --mode fast
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test

	diff -Nau guided-setup.postgres.expected guided-setup.postgres.out && echo 'postgres configuration modified as expected'
	grep -v db_password guided-setup.collector-config.out > guided-setup.collector-config.nopass.out
	diff -Nau guided-setup.collector-config.nopass.expected guided-setup.collector-config.nopass.out \
		&& grep --extended-regexp --quiet 'db_password\s+=\s+[a-f0-9]{32}' guided-setup.collector-config.out && echo 'collector config file modified as expected'
	jq '{fns: .functionReferences, roles: .roleReferences[] |  select(.name == "pganalyze")}' < guided-setup.snapshot.json.out > guided-setup.snapshot-subset.json.out
	if [ "`grep 'Local log test successful' guided-setup-snapshot.log.out | wc -l`" -ne 2 ]; then \
		echo "expected post-setup log test to succeed with both full and reduced privileges; test failed:"; \
		cat guided-setup-snapshot.log.out; \
		exit 1; \
	fi
	diff -Nau guided-setup.snapshot-subset.json.expected guided-setup.snapshot-subset.json.out && echo 'success'

installer:
	docker build -f Dockerfile.test-guided-setup $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10

	docker exec --privileged pganalyze-collector-test env PGA_INSTALL_NONINTERACTIVE=true PGA_API_KEY=abc123 bash -c '</collector/contrib/install.sh bash' >install.out
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test

	if ! grep -q 'The pganalyze collector was installed successfully' install.out; then \
		echo "expected installer to install latest published package; test failed:"; \
		cat install.out; \
		exit 1; \
	fi
