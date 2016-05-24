OUTFILE := collector
PROTOBUF_FILE = snapshot.proto

default: prepare build test

prepare:
	go get -d
	make -C ${GOPATH}/src/github.com/lfittl/pg_query_go build
	go get

snapshot/snapshot.pb.go: $(PROTOBUF_FILE)
	protoc --go_out=Mgoogle/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp:snapshot $(PROTOBUF_FILE)

build: snapshot/snapshot.pb.go
	go build -o ${OUTFILE}

test: build
	go test -v ./ ./dbstats ./scheduler

.PHONY: test
