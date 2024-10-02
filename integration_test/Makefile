DOCKER_BUILD_OPTS = -t pganalyze-collector-test ..

# See this on running systemd inside containers with CgroupsV2:
# - https://systemd.io/CONTAINER_INTERFACE/
# - https://serverfault.com/questions/1053187/systemd-fails-to-run-in-a-docker-container-when-using-cgroupv2-cgroupns-priva
# - Check if your system supports Cgroupsv2 by testing presence of /sys/fs/cgroup/cgroup.controllers
docker_run_cmd = docker run --name pganalyze-collector-test \
  --cgroup-parent=docker.slice --cgroupns private \
  --privileged --cap-add SYS_ADMIN --security-opt seccomp=unconfined \
  --tmpfs /tmp --tmpfs /run --tmpfs /run/lock \
  -v `pwd`/docker-entrypoint-initdb.d:/docker-entrypoint-initdb.d/ \
  -e POSTGRES_HOST_AUTH_METHOD=trust \
  -d pganalyze-collector-test $(1)
docker_run_cmd_postgres = $(call docker_run_cmd,postgres -c pg_stat_statements.track_utility=off)

TARGETS := pg10 pg11 pg12 pg13 pg14 pg15 pg16 reload guided-setup installer

# Citus doesn't release ARM images, thus skip on ARM
ifeq ($(findstring $(shell uname -m),arm64 aarch64),)
	TARGETS += citus
endif

.PHONY: all $(TARGETS)

all: $(TARGETS)

pg10:
	docker build -f Dockerfile.test-pg10 $(DOCKER_BUILD_OPTS)
	$(docker_run_cmd_postgres)
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
	$(docker_run_cmd_postgres)
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
	$(docker_run_cmd_postgres)
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
	$(docker_run_cmd_postgres)
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

pg14:
	docker build -f Dockerfile.test-pg14 $(DOCKER_BUILD_OPTS)
	$(docker_run_cmd_postgres)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg14_1.json
	docker exec pganalyze-collector-test pgbench -U postgres postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg14_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg14_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg14_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg14_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg14_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg14_2.json | sort --version-sort -f > pg14.out
	diff -Naur pg14.expected pg14.out && echo 'success'

pg15:
	docker build -f Dockerfile.test-pg15 $(DOCKER_BUILD_OPTS)
	$(docker_run_cmd_postgres)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg15_1.json
	docker exec pganalyze-collector-test pgbench -U postgres postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg15_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg15_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg15_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg15_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg15_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg15_2.json | sort --version-sort -f > pg15.out
	diff -Naur pg15.expected pg15.out && echo 'success'

pg16:
	docker build -f Dockerfile.test-pg16 $(DOCKER_BUILD_OPTS)
	$(docker_run_cmd_postgres)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg16_1.json
	docker exec pganalyze-collector-test pgbench -U postgres postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg16_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg16_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg16_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg16_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg16_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg16_2.json | sort --version-sort -f > pg16.out
	diff -Naur pg16.expected pg16.out && echo 'success'

pg17:
	docker build -f Dockerfile.test-pg17 $(DOCKER_BUILD_OPTS)
	$(docker_run_cmd_postgres)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg17_1.json
	docker exec pganalyze-collector-test pgbench -U postgres postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > pg17_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' pg17_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' pg17_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' pg17_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' pg17_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' pg17_2.json | sort --version-sort -f > pg17.out
	diff -Naur pg17.expected pg17.out && echo 'success'

citus:
	docker build -f Dockerfile.test-citus $(DOCKER_BUILD_OPTS)
	$(docker_run_cmd_postgres)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres postgres -i
	docker exec -i pganalyze-collector-test psql -U postgres postgres < citus-pgbench-ddl.sql
	docker exec pganalyze-collector-test pgbench -U postgres postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > citus_1.json
	docker exec pganalyze-collector-test pgbench -U postgres postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy PGA_DISABLE_ACTIVITY=1 IGNORE_SCHEMA_REGEXP=pgbench_history pganalyze-collector --dry-run --force-state-update -v" > citus_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
	if [ "`jq '.collectorErrors' citus_2.json | tr -d ' \n\r\t '`" != "null" ]; then \
		jq '.collectorErrors' citus_2.json; \
		exit 1; \
	fi
	if jq '.relationReferences' citus_2.json | grep --quiet pgbench_history; then \
                echo "failed to ignore table according to IGNORE_SCHEMA_REGEXP=pgbench_history" \
		jq '.relationReferences' citus_2.json; \
		exit 1; \
	fi
	jq '.queryInformations[].normalizedQuery' citus_2.json | sed -E 's/pgbench_branches_[0-9]+/pgbench_branches_XXXXX/' | sort --version-sort -f > citus.out
	jq '.relationReferences[].relationName' citus_2.json | sort >> citus.out
	diff -Naur citus.expected citus.out && echo 'success'

# Test reload both with an empty config and a server present
#
# Note that we reload twice after adding a server to the config, so we can test the
# "write state file when reload occurs" logic, that requires the server to be there
# *before* the reload
reload:
	docker build -f Dockerfile.test-reload $(DOCKER_BUILD_OPTS)
	$(call docker_run_cmd,)
	sleep 5
	docker exec pganalyze-collector-test systemctl start pganalyze-collector
	docker exec pganalyze-collector-test pganalyze-collector --reload
	docker exec pganalyze-collector-test journalctl -u pganalyze-collector | grep "Reloading configuration..."
	docker exec pganalyze-collector-test test \! -f /var/lib/pganalyze-collector/state
	docker exec pganalyze-collector-test bash -c "echo 'db_url = postgres://127.0.0.1/postgres' >> /etc/pganalyze-collector.conf"
	docker exec pganalyze-collector-test pganalyze-collector --reload
	docker exec pganalyze-collector-test pganalyze-collector --reload
	docker exec pganalyze-collector-test test -f /var/lib/pganalyze-collector/state
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test

guided-setup:
	docker build -f Dockerfile.test-guided-setup $(DOCKER_BUILD_OPTS)
	$(call docker_run_cmd,)
	sleep 5
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

	docker exec --user postgres pganalyze-collector-test pg_ctl --pgdata /var/lib/postgresql/12/main --wait stop --mode fast
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test

	diff -Nau guided-setup.postgres.expected guided-setup.postgres.out && echo 'postgres configuration modified as expected'
	grep -v db_password guided-setup.collector-config.out > guided-setup.collector-config.nopass.out
	diff -Nau guided-setup.collector-config.nopass.expected guided-setup.collector-config.nopass.out \
		&& grep --extended-regexp --quiet 'db_password\s+=\s+[a-f0-9]{32}' guided-setup.collector-config.out && echo 'collector config file modified as expected'
	jq '{roles: .roleReferences[] |  select(.name == "pganalyze")}' < guided-setup.snapshot.json.out > guided-setup.snapshot-subset.json.out
	if [ "`grep 'Local log test successful' guided-setup-snapshot.log.out | wc -l`" -ne 2 ]; then \
		echo "expected post-setup log test to succeed with both full and reduced privileges; test failed:"; \
		cat guided-setup-snapshot.log.out; \
		exit 1; \
	fi
	if grep -q -E '[0-9]{4}\/[0-9]{2}\/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} W ' guided-setup-snapshot.log.out; then \
		echo "expected no warnings in collector log; test failed:"; \
		cat guided-setup-snapshot.log.out; \
		exit 1; \
	fi
	diff -Nau guided-setup.snapshot-subset.json.expected guided-setup.snapshot-subset.json.out && echo 'success'

installer:
	docker build -f Dockerfile.test-guided-setup $(DOCKER_BUILD_OPTS)
	$(call docker_run_cmd,)
	sleep 5
	docker exec pganalyze-collector-test systemctl start postgresql

	docker exec --privileged pganalyze-collector-test env PGA_INSTALL_NONINTERACTIVE=true PGA_API_KEY=abc123 bash -c '</collector/contrib/install.sh bash' >install.out
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test

	if ! grep -q 'The pganalyze collector was installed successfully' install.out; then \
		echo "expected installer to install latest published package; test failed:"; \
		cat install.out; \
		exit 1; \
	fi
