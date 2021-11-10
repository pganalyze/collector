OUTFILE := pganalyze-collector
PROTOBUF_FILES := $(wildcard protobuf/*.proto) $(wildcard protobuf/reports/*.proto)

PATH := $(PWD)/protoc/bin:$(PWD)/bin:$(PATH)
SHELL := env PATH=$(PATH) /bin/sh

PROTOC_VERSION_NEEDED := 3.14.0
PROTOC_VERSION := $(shell which protoc && protoc --version)

.PHONY: default build build_dist vendor test docker_latest packages integration_test

default: build test

build: install_protoc output/pganalyze_collector/snapshot.pb.go build_dist

build_dist:
	go build -o ${OUTFILE}
	make -C helper OUTFILE=../pganalyze-collector-helper
	make -C setup OUTFILE=../pganalyze-collector-setup

build_dist_alpine:
	# Increase stack size from Alpine's default of 80kb to 2mb - otherwise we see
	# crashes on very complex queries, pg_query expects at least 100kb stack size
	go build -o ${OUTFILE} -ldflags '-extldflags "-Wl,-z,stack-size=0x200000"'
	make -C helper OUTFILE=../pganalyze-collector-helper
	make -C setup OUTFILE=../pganalyze-collector-setup

vendor:
	GO111MODULE=on go mod tidy
	# See CONTRIBUTING.md if modvendor can't be found
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
	protoc --go_out=Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp:output/pganalyze_collector -I protobuf $(PROTOBUF_FILES)
else
	@echo 'Warning: protoc not found, skipping protocol buffer regeneration (to install protoc check Makefile instructions in install_protoc step)'
endif

install_protoc:
ifdef PROTOC_VERSION
ifeq (,$(findstring $(PROTOC_VERSION_NEEDED), $(PROTOC_VERSION)))
	@echo "⚠️  protoc version needed: $(PROTOC_VERSION_NEEDED) vs $(PROTOC_VERSION) installed"
	@echo "ℹ️  Please download the correct protobuf binary for your OS from https://github.com/protocolbuffers/protobuf/releases/tag/v${PROTOC_VERSION_NEEDED}"
	@echo "ℹ️  Note the download's name will look like this: protoc-${PROTOC_VERSION_NEEDED}-osx-x86_64.zip"
	@echo "ℹ️  Copy the unzipped folder into this project, and rename it to \"protoc\""
	@echo "ℹ️  If this is macOS, you will need to try running the binary yourself, then go to Security & Privacy to explicitly allow it."
	exit 1
endif
endif
