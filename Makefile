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

release: test
	docker build -t quay.io/pganalyze/collector:protobuf .
	docker push quay.io/pganalyze/collector:protobuf

.PHONY: default prepare build test release
