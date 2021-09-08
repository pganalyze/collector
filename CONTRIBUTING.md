## Contributing Instructions

### Setup

The dependencies are stored in the `vendor` folder, so no installation is needed.

### Updating dependencies

```sh
go get -u github.com/goware/modvendor
make vendor
```

### Compiling and running tests

To compile the collector and helper binaries:

```sh
make build
```

After building the collector you can find the binary in the repository folder:

```sh
./pganalyze-collector --help
```

To run the unit tests:

```sh
make test
```

To run the integration tests:

```sh
make integration_test
```

Note the integration tests require Docker, and will take a while to run through.
