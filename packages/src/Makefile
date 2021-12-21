# Note: This requires variables that are set in the top-level packages Makefile

BUILD_ARGS=--no-cache --build-arg VERSION=$(VERSION) --build-arg GIT_VERSION=$(GIT_VERSION)

.PHONY: all

all: $(TMP_DIR)/$(RPM_PACKAGE_X86_64) $(TMP_DIR)/$(RPM_PACKAGE_ARM64) \
$(TMP_DIR)/$(DEB_PACKAGE_X86_64) $(TMP_DIR)/$(DEB_PACKAGE_ARM64)

$(TMP_DIR)/$(RPM_PACKAGE_X86_64): Dockerfile.build.rpm-systemd
	docker build --platform linux/amd64 $(BUILD_ARGS) -f Dockerfile.build.rpm-systemd -t pga-collector-build ../../
	docker run --platform linux/amd64 --rm -v $(TMP_DIR):/out -it pga-collector-build sh -c "cp /root/$(RPM_PACKAGE_X86_64) /out"
	docker rmi pga-collector-build

$(TMP_DIR)/$(RPM_PACKAGE_ARM64): Dockerfile.build.rpm-systemd
	docker build --platform linux/arm64 $(BUILD_ARGS) -f Dockerfile.build.rpm-systemd -t pga-collector-build ../../
	docker run --platform linux/arm64 --rm -v $(TMP_DIR):/out -it pga-collector-build sh -c "cp /root/$(RPM_PACKAGE_ARM64) /out"
	docker rmi pga-collector-build

$(TMP_DIR)/$(DEB_PACKAGE_X86_64): Dockerfile.build.deb-systemd
	docker build --platform linux/amd64 $(BUILD_ARGS) -f Dockerfile.build.deb-systemd -t pga-collector-build ../../
	docker run --platform linux/amd64 --rm -v $(TMP_DIR):/out -it pga-collector-build sh -c "cp /root/$(DEB_PACKAGE_X86_64) /out"
	docker rmi pga-collector-build

$(TMP_DIR)/$(DEB_PACKAGE_ARM64): Dockerfile.build.deb-systemd
	docker build --platform linux/arm64 $(BUILD_ARGS) -f Dockerfile.build.deb-systemd -t pga-collector-build ../../
	docker run --platform linux/arm64 --rm -v $(TMP_DIR):/out -it pga-collector-build sh -c "cp /root/$(DEB_PACKAGE_ARM64) /out"
	docker rmi pga-collector-build
