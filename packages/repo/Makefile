# Note: This requires variables that are set in the top-level packages Makefile

REPO_GPG_KEY=A2B5F2F9 # If you change this also update deb.distributions
REPO_DIR=$(TMP_DIR)/repo

docker_clean = docker kill pga-collector-repo && docker rm pga-collector-repo && docker rmi -f pga-collector-repo

.PHONY: update_rpm_repo push_packages_release push_packages_latest

all: download_repo update_rpm update_deb upload_repo

download_repo:
	rm -rf $(REPO_DIR)
	aws s3 sync s3://packages.pganalyze.com/ $(REPO_DIR)
	mkdir -p $(REPO_DIR)

upload_repo:
	aws s3 sync --acl public-read --cache-control no-cache $(REPO_DIR) s3://packages.pganalyze.com/

update_rpm:
	cp $(TMP_DIR)/$(RPM_SYSVINIT_PACKAGE) .
	cp $(TMP_DIR)/$(RPM_SYSTEMD_PACKAGE) .
	docker build --build-arg VERSION=$(VERSION) -f Dockerfile.repo-rpm -t pga-collector-repo .
	rm $(RPM_SYSVINIT_PACKAGE)
	rm $(RPM_SYSTEMD_PACKAGE)
	docker run --name pga-collector-repo -v $(REPO_DIR):/repo -d \
		-e REPO_GPG_KEY=$(REPO_GPG_KEY) -e RPM_PACKAGE=$(RPM_PACKAGE_OUT) \
		pga-collector-repo /sbin/init
	keybase pgp export --unencrypted -s -q $(REPO_GPG_KEY) | docker exec -i pga-collector-repo gpg --allow-secret-key-import --import
	docker exec -i pga-collector-repo /root/sync_rpm.sh
	$(call docker_clean)

update_deb:
	cp $(TMP_DIR)/$(DEB_UPSTART_PACKAGE) .
	cp $(TMP_DIR)/$(DEB_SYSTEMD_PACKAGE) .
	docker build --build-arg VERSION=$(VERSION) -f Dockerfile.repo-deb -t pga-collector-repo .
	rm $(DEB_UPSTART_PACKAGE)
	rm $(DEB_SYSTEMD_PACKAGE)
	docker run --name pga-collector-repo -v $(REPO_DIR):/repo -d \
		-e REPO_GPG_KEY=$(REPO_GPG_KEY) -e DEB_PACKAGE=$(DEB_PACKAGE_OUT) \
		pga-collector-repo /sbin/init
	keybase pgp export --unencrypted -s -q $(REPO_GPG_KEY) | docker exec -i pga-collector-repo gpg --allow-secret-key-import --import
	docker exec -i pga-collector-repo /root/sync_deb.sh
	$(call docker_clean)
