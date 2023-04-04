export NAME ?= pganalyze-collector
export VERSION ?= 0.49.1
export GIT_VERSION ?= v$(VERSION)

export RPM_PACKAGE_X86_64=$(NAME)-$(VERSION)-1.x86_64.rpm
export RPM_PACKAGE_ARM64=$(NAME)-$(VERSION)-1.aarch64.rpm
export DEB_PACKAGE_X86_64=$(NAME)_$(VERSION)_amd64.deb
export DEB_PACKAGE_ARM64=$(NAME)_$(VERSION)_arm64.deb

export TMP_DIR=$(shell pwd)/tmp

.PHONY: default setup build test clean repo

default: test repo

setup:
	mkdir -p $(TMP_DIR)

build: setup
	make -C src

test: setup build
	make -C test

repo: build
	make -C repo

clean:
	rm -rf $(TMP_DIR)
