## Contributing Instructions

### Setup

```sh
# install golang, then:
go get -u github.com/goware/modvendor
make vendor
```

### Compiling and running tests

```sh
make build
make test
# see the others in the Makefile

./pganalyze-collector --help
```
