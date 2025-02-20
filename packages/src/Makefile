# Note: This requires variables that are set in the top-level packages Makefile

BUILD_ARGS=--no-cache --build-arg VERSION=$(VERSION) --build-arg GIT_VERSION=$(GIT_VERSION)
DOCKER_CMD=docker

.PHONY: all x86_64 arm64

all: x86_64 arm64

x86_64: $(TMP_DIR)/$(RPM_PACKAGE_X86_64) $(TMP_DIR)/$(DEB_PACKAGE_X86_64)

arm64: $(TMP_DIR)/$(RPM_PACKAGE_ARM64) $(TMP_DIR)/$(DEB_PACKAGE_ARM64)

$(TMP_DIR)/$(RPM_PACKAGE_X86_64): Dockerfile.build.rpm-systemd
	$(DOCKER_CMD) build --platform linux/amd64 $(BUILD_ARGS) -f Dockerfile.build.rpm-systemd -t pga-collector-build ../../
	$(DOCKER_CMD) run --platform linux/amd64 --rm -v $(TMP_DIR):/out pga-collector-build sh -c "cp /root/$(RPM_PACKAGE_X86_64) /out"
	$(DOCKER_CMD) rmi pga-collector-build

$(TMP_DIR)/$(RPM_PACKAGE_ARM64): Dockerfile.build.rpm-systemd
	$(DOCKER_CMD) build --platform linux/arm64 $(BUILD_ARGS) -f Dockerfile.build.rpm-systemd -t pga-collector-build ../../
	$(DOCKER_CMD) run --platform linux/arm64 --rm -v $(TMP_DIR):/out pga-collector-build sh -c "cp /root/$(RPM_PACKAGE_ARM64) /out"
	$(DOCKER_CMD) rmi pga-collector-build

$(TMP_DIR)/$(DEB_PACKAGE_X86_64): Dockerfile.build.deb-systemd
	$(DOCKER_CMD) build --platform linux/amd64 $(BUILD_ARGS) -f Dockerfile.build.deb-systemd -t pga-collector-build ../../
	$(DOCKER_CMD) run --platform linux/amd64 --rm -v $(TMP_DIR):/out pga-collector-build sh -c "cp /root/$(DEB_PACKAGE_X86_64) /out"
	$(DOCKER_CMD) rmi pga-collector-build

$(TMP_DIR)/$(DEB_PACKAGE_ARM64): Dockerfile.build.deb-systemd
	$(DOCKER_CMD) build --platform linux/arm64 $(BUILD_ARGS) -f Dockerfile.build.deb-systemd -t pga-collector-build ../../
	$(DOCKER_CMD) run --platform linux/arm64 --rm -v $(TMP_DIR):/out pga-collector-build sh -c "cp /root/$(DEB_PACKAGE_ARM64) /out"
	$(DOCKER_CMD) rmi pga-collector-build
