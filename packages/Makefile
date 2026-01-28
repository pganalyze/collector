export NAME ?= pganalyze-collector
export VERSION ?= 0.68.1
export GIT_VERSION ?= v$(VERSION)
#export GIT_VERSION=HEAD
#export GIT_VERSION=618e85ce5ed5365bc7d9d9da866fdeb73bac5a55
#export VERSION=$(shell git show -s --format=%ct.%h HEAD)

export RPM_PACKAGE_X86_64=$(NAME)-$(VERSION)-1.x86_64.rpm
export RPM_PACKAGE_ARM64=$(NAME)-$(VERSION)-1.aarch64.rpm
export DEB_PACKAGE_X86_64=$(NAME)_$(VERSION)_amd64.deb
export DEB_PACKAGE_ARM64=$(NAME)_$(VERSION)_arm64.deb

export TMP_DIR=$(shell pwd)/tmp

.PHONY: default setup build_x86_64 build_arm64 build test_x86_64 test_arm64 test clean repo

default: test repo

setup:
	mkdir -p $(TMP_DIR)

build_x86_64: setup
	make -C src x86_64

build_arm64: setup
	make -C src arm64

build: build_x86_64 build_arm64

test_x86_64: setup build_x86_64
	make -C test x86_64

test_arm64: setup build_arm64
	make -C test arm64

test: test_x86_64 test_arm64

# Assume the package build and test were run automatically, and use the artifacts from the GitHub release instead of rebuilding
repo: setup
	curl -o tmp/${RPM_PACKAGE_X86_64} -fSL "https://github.com/pganalyze/collector/releases/download/${GIT_VERSION}/${RPM_PACKAGE_X86_64}"
	curl -o tmp/${RPM_PACKAGE_ARM64} -fSL "https://github.com/pganalyze/collector/releases/download/${GIT_VERSION}/${RPM_PACKAGE_ARM64}"
	curl -o tmp/${DEB_PACKAGE_X86_64} -fSL "https://github.com/pganalyze/collector/releases/download/${GIT_VERSION}/${DEB_PACKAGE_X86_64}"
	curl -o tmp/${DEB_PACKAGE_ARM64} -fSL "https://github.com/pganalyze/collector/releases/download/${GIT_VERSION}/${DEB_PACKAGE_ARM64}"
	make -C repo

clean:
	rm -rf $(TMP_DIR)
