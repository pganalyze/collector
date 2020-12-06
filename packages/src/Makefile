# Note: This requires variables that are set in the top-level packages Makefile

TMP_RPM_SYSTEMD = $(TMP_DIR)/build-rpm-systemd
TMP_DEB_SYSTEMD = $(TMP_DIR)/build-deb-systemd

BUILD_ARGS=--build-arg VERSION=$(VERSION) --build-arg GIT_VERSION=$(GIT_VERSION)

.PHONY: all

all: $(TMP_DIR)/$(RPM_SYSTEMD_PACKAGE) $(TMP_DIR)/$(DEB_SYSTEMD_PACKAGE)

$(TMP_DIR)/$(RPM_SYSTEMD_PACKAGE):
	docker build $(BUILD_ARGS) -f Dockerfile.build.rpm-systemd -t pga-collector-build ../../
	docker run --rm -v $(TMP_RPM_SYSTEMD):/out -it pga-collector-build sh -c "cp /root/*.rpm /out"
	mv $(TMP_RPM_SYSTEMD)/$(RPM_PACKAGE_OUT) $(TMP_DIR)/$(RPM_SYSTEMD_PACKAGE)
	docker rmi pga-collector-build

$(TMP_DIR)/$(DEB_SYSTEMD_PACKAGE):
	docker build $(BUILD_ARGS) -f Dockerfile.build.deb-systemd -t pga-collector-build ../../
	docker run --rm -v $(TMP_DEB_SYSTEMD):/out -it pga-collector-build sh -c "cp /root/*.deb /out"
	mv $(TMP_DEB_SYSTEMD)/$(DEB_PACKAGE_OUT) $(TMP_DIR)/$(DEB_SYSTEMD_PACKAGE)
	docker rmi pga-collector-build
