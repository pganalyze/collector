OUTFILE := pganalyze-collector
PROTOBUF_FILES := $(wildcard protobuf/*.proto) $(wildcard protobuf/reports/*.proto)
PROTOC_VERSION := $(shell protoc --version 2>/dev/null)

.PHONY: default build build_dist vendor test docker_latest packages integration_test

default: build test

build: output/pganalyze_collector/snapshot.pb.go build_dist

build_dist:
	go build -o ${OUTFILE}
	make -C helper OUTFILE=../pganalyze-collector-helper

vendor:
	GO111MODULE=on go mod vendor
	# You might need to run "go get -u github.com/goware/modvendor"
	modvendor -copy="**/*.c **/*.h **/*.proto" -v

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
ifdef PROTOC_VERSION
	protoc --go_out=Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp:output/pganalyze_collector -I protobuf $(PROTOBUF_FILES)
else
	@echo 'Warning: protoc not found, skipping protocol buffer regeneration'
endif
