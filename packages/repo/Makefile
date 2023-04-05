# Note: This requires variables that are set in the top-level packages Makefile

REPO_GPG_KEY=A2B5F2F9 # If you change this also update deb.distributions
REPO_DIR=$(TMP_DIR)/repo
DOCKER_CMD=docker

docker_clean = $(DOCKER_CMD) kill pga-collector-repo && $(DOCKER_CMD) rm pga-collector-repo && $(DOCKER_CMD) rmi -f pga-collector-repo

.PHONY: all download_repo upload_repo copy_install_script update_rpm update_deb

all: download_repo update_rpm update_deb copy_install_script upload_repo

download_repo:
	rm -rf $(REPO_DIR)
	aws s3 sync s3://packages.pganalyze.com/ $(REPO_DIR)
	mkdir -p $(REPO_DIR)

upload_repo:
	aws s3 sync --acl public-read --cache-control no-cache $(REPO_DIR) s3://packages.pganalyze.com/

copy_install_script:
	cp ../../contrib/install.sh $(REPO_DIR)/collector-install.sh

update_rpm:
	cp $(TMP_DIR)/$(RPM_PACKAGE_X86_64) .
	cp $(TMP_DIR)/$(RPM_PACKAGE_ARM64) .
	$(DOCKER_CMD) build --build-arg VERSION=$(VERSION) -f Dockerfile.repo-rpm -t pga-collector-repo .
	rm $(RPM_PACKAGE_X86_64)
	rm $(RPM_PACKAGE_ARM64)
	$(DOCKER_CMD) run --name pga-collector-repo -v $(REPO_DIR):/repo -d \
		-e REPO_GPG_KEY=$(REPO_GPG_KEY) \
		-e RPM_PACKAGE_X86_64=$(RPM_PACKAGE_X86_64) \
		-e RPM_PACKAGE_ARM64=$(RPM_PACKAGE_ARM64) \
		pga-collector-repo /sbin/init
	keybase pgp export --unencrypted -s -q $(REPO_GPG_KEY) | $(DOCKER_CMD) exec -i pga-collector-repo gpg --allow-secret-key-import --import
	$(DOCKER_CMD) exec -i pga-collector-repo /root/sync_rpm.sh
	$(call docker_clean)

update_deb:
	cp $(TMP_DIR)/$(DEB_PACKAGE_X86_64) .
	cp $(TMP_DIR)/$(DEB_PACKAGE_ARM64) .
	$(DOCKER_CMD) build --build-arg VERSION=$(VERSION) -f Dockerfile.repo-deb -t pga-collector-repo .
	rm $(DEB_PACKAGE_X86_64)
	rm $(DEB_PACKAGE_ARM64)
	$(DOCKER_CMD) run --name pga-collector-repo -v $(REPO_DIR):/repo -d \
		-e REPO_GPG_KEY=$(REPO_GPG_KEY) \
		-e DEB_PACKAGE_X86_64=$(DEB_PACKAGE_X86_64) \
		-e DEB_PACKAGE_ARM64=$(DEB_PACKAGE_ARM64) \
		-e container=docker \
		--privileged=true \
		pga-collector-repo /sbin/init
	keybase pgp export --unencrypted -s -q $(REPO_GPG_KEY) | $(DOCKER_CMD) exec -i pga-collector-repo gpg --allow-secret-key-import --import
	$(DOCKER_CMD) exec -i pga-collector-repo /root/sync_deb.sh
	$(call docker_clean)
