OUTFILE := collector
PROTOBUF_FILE = snapshot.proto

default: prepare build test

prepare: output/pganalyze_collector/snapshot.pb.go
	go get

output/pganalyze_collector/snapshot.pb.go: $(PROTOBUF_FILE)
	protoc --go_out=Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp:output/pganalyze_collector $(PROTOBUF_FILE)

build: output/pganalyze_collector/snapshot.pb.go
	go build -o ${OUTFILE}

test: build
	go test -v ./ ./scheduler ./util ./output/transform/

packages:
	make -C packages

release_latest: test
	docker build -t quay.io/pganalyze/collector:latest .
	docker push quay.io/pganalyze/collector:latest
	make -C packages push_packages_latest

.PHONY: default prepare build test release_latest packages
