OUTFILE := collector
PROTOBUF_FILE = snapshot.proto

default: prepare build test

prepare: output/snapshot/snapshot.pb.go
	go get -d
	make -C ${GOPATH}/src/github.com/lfittl/pg_query_go build
	go get

output/snapshot/snapshot.pb.go: $(PROTOBUF_FILE)
	protoc --go_out=Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp:output/snapshot $(PROTOBUF_FILE)

build: output/snapshot/snapshot.pb.go
	go build -o ${OUTFILE}

test: build
	go test -v ./ ./scheduler

.PHONY: test
