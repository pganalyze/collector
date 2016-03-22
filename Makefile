default: build test

build:
	go get -d
	make -C ${GOPATH}/src/github.com/lfittl/pg_query_go build
	go get
	go build -o pganalyze-collector

test: build
	go test -v ./ ./dbstats ./scheduler

.PHONY: test
