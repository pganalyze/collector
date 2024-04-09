## Contributing Instructions

Pull the project with submodules.

```sh
git clone --recursive https://github.com/pganalyze/collector
```

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
3. Once a new tag is pushed, GitHub Action Release will be kicked and create a new release (this will take about 2 hours, due to the package build and test)
4. Modify the newly created release's description to match to CHANGELOG.md
5. Release docker images using `make docker_release` (this requires access to the Quay.io push key, as well as "docker buildx" with QEMU emulation support, see below)
6. Sign and release packages using `make -C packages repo` (this requires access to the Keybase GPG key)

To run step 5 from an Ubuntu 22.04 VM, do the following:

```
# Add Docker's official GPG key:
sudo apt-get update
sudo apt-get install ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# Add the repository to Apt sources:
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin

# Add support for ARM emulation
sudo apt update
sudo apt install qemu-user-static binfmt-support make

# Get these credentials from Quay.io
sudo docker login -u="REPLACE_ME" -p="REPLACE_ME" quay.io

git clone https://github.com/pganalyze/collector.git
cd collector
sudo make docker_release
```
