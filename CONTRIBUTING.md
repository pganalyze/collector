## Contributing Instructions

### Setup

The dependencies are stored in the `vendor` folder, so no installation is needed.

#### Setup for updating dependencies

```sh
go install github.com/goware/modvendor@latest
```

Then, make sure Go package executables are on your `$PATH`. For Homebrew on macOS that is `~/go/bin`. If it's working, `which modvendor` should return the path that modvendor is installed at.

```sh
make vendor
```

### Updating dependencies

```sh
go get github.com/shirou/gopsutil@latest # updates the version requirement
make vendor                              # updates the vendored code
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

### Release

1. Create a PR to update the version numbers and CHANGELOG.md
2. Once PR is merged, create a new tag `git tag v0.x.y`, then push it `git push origin v0.x.y`
3. Once a new tag is pushed, GitHub Action Release will be kicked and create a new release
4. Modify a newly created release's description to match to CHANGELOG.md