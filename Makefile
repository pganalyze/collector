OUTFILE := pganalyze-collector
PROTOBUF_FILES := $(wildcard protobuf/*.proto) $(wildcard protobuf/reports/*.proto)

.PHONY: default build build_dist test docker_latest packages integration_test

default: build test

build: output/pganalyze_collector/snapshot.pb.go build_dist

build_dist:
	ulimit -n 2048 # https://github.com/golang/go/issues/21621
	go build -o ${OUTFILE}
	make -C helper OUTFILE=../pganalyze-collector-helper

test: build
	go test -v ./ ./scheduler ./util ./output/transform/ ./input/system/logs/

integration_test:
	make -C integration_test

packages:
	make -C packages

docker_latest:
	docker build -t quay.io/pganalyze/collector:latest .
	docker push quay.io/pganalyze/collector:latest

output/pganalyze_collector/snapshot.pb.go: $(PROTOBUF_FILES)
	protoc --go_out=Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp:output/pganalyze_collector -I protobuf $(PROTOBUF_FILES)
