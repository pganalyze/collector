DOCKER_BUILD_OPTS = -t pganalyze-collector-test ..
DOCKER_RUN_CMD = docker run --name pganalyze-collector-test -v `pwd`/docker-entrypoint-initdb.d:/docker-entrypoint-initdb.d/ -d pganalyze-collector-test

.PHONY: pg92 pg93 pg94 pg95 pg96

all: pg92 pg93 pg94 pg95 pg96

pg92:
	docker build -f Dockerfile.test-pg92 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy pganalyze-collector --dry-run --force-state-update -v" > pg92_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy pganalyze-collector --dry-run --force-state-update -v" > pg92_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test

pg93:
	docker build -f Dockerfile.test-pg93 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy pganalyze-collector --dry-run --force-state-update -v" > pg93_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy pganalyze-collector --dry-run --force-state-update -v" > pg93_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test

pg94:
	docker build -f Dockerfile.test-pg94 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy pganalyze-collector --dry-run --force-state-update -v" > pg94_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy pganalyze-collector --dry-run --force-state-update -v" > pg94_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test

pg95:
	docker build -f Dockerfile.test-pg95 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy pganalyze-collector --dry-run --force-state-update -v" > pg95_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy pganalyze-collector --dry-run --force-state-update -v" > pg95_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test

pg96:
	docker build -f Dockerfile.test-pg96 $(DOCKER_BUILD_OPTS)
	$(DOCKER_RUN_CMD)
	sleep 10
	docker exec pganalyze-collector-test pgbench -U postgres -i
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy pganalyze-collector --dry-run --force-state-update -v" > pg96_1.json
	docker exec pganalyze-collector-test pgbench -U postgres
	docker exec pganalyze-collector-test sh -c "DB_USERNAME=postgres PGA_API_KEY=dummy pganalyze-collector --dry-run --force-state-update -v" > pg96_2.json
	docker rm -f pganalyze-collector-test
	docker rmi pganalyze-collector-test
