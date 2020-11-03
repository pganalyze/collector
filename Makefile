OUTFILE := pganalyze-collector
PROTOBUF_FILES := $(wildcard protobuf/*.proto) $(wildcard protobuf/reports/*.proto)
PROTOC_VERSION := $(shell protoc --version 2>/dev/null)

.PHONY: default build build_dist vendor test docker_latest packages integration_test

default: build test

build: output/pganalyze_collector/snapshot.pb.go build_dist

build_dist:
	go build -o ${OUTFILE}
	make -C helper OUTFILE=../pganalyze-collector-helper
	make -C setup OUTFILE=../pganalyze-collector-setup

vendor:
	GO111MODULE=on go mod tidy
	# You might need to run "go get -u github.com/goware/modvendor"
	GO111MODULE=on go mod vendor
	modvendor -copy="**/*.c **/*.h **/*.proto" -v

test: build
	go test -coverprofile=coverage.out ./...
	# go tool cover -html=coverage.out

integration_test:
	make -C integration_test

packages:
	make -C packages

docker_latest:
	docker build -t quay.io/pganalyze/collector:latest .
	docker push quay.io/pganalyze/collector:latest

output/pganalyze_collector/snapshot.pb.go: $(PROTOBUF_FILES)
ifdef PROTOC_VERSION
	mkdir -p $(PWD)/bin
	GOBIN=$(PWD)/bin go install github.com/golang/protobuf/protoc-gen-go
	PATH=$(PWD)/bin:$(PATH) protoc --go_out=Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp:output/pganalyze_collector -I protobuf $(PROTOBUF_FILES)
else
	@echo 'Warning: protoc not found, skipping protocol buffer regeneration'
endif
