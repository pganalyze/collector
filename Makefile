OUTFILE := pganalyze-collector
PROTOBUF_FILES := $(wildcard protobuf/*.proto)

PATH := $(PWD)/protoc/bin:$(PWD)/bin:$(PATH)
SHELL := env PATH=$(PATH) /bin/sh

PROTOC_VERSION_NEEDED := 28.2
PROTOC_VERSION := $(shell command -v protoc > /dev/null 2>&1 && protoc --version || echo 'none')
PROTOC_BASE_URL := https://github.com/protocolbuffers/protobuf/releases/download/

ARCH := $(shell uname -m)
ifneq (,$(filter $(ARCH),arm64 aarch64))
	PROTOC_ARCH := aarch_64
else
	PROTOC_ARCH := $(ARCH)
endif

ifeq ($(shell uname), Linux)
	PROTOC_FILENAME := protoc-$(PROTOC_VERSION_NEEDED)-linux-$(PROTOC_ARCH).zip
endif
ifeq ($(shell uname), Darwin)
	PROTOC_FILENAME := protoc-$(PROTOC_VERSION_NEEDED)-osx-$(PROTOC_ARCH).zip
endif
PROTOC_URL := $(PROTOC_BASE_URL)v$(PROTOC_VERSION_NEEDED)/$(PROTOC_FILENAME)

.PHONY: default build build_dist vendor test docker_image_amd64 docker_image_arm64 docker_release_staging docker_release_manifest packages integration_test

default: build test

build: install_protoc output/pganalyze_collector/snapshot.pb.go build_dist

build_dist:
	go build -o ${OUTFILE}
	make -C helper OUTFILE=../pganalyze-collector-helper
	make -C setup OUTFILE=../pganalyze-collector-setup
	# Sign built Go binaries for macOS
	@if [ "$(shell uname)" = "Darwin" ] && command -v codesign > /dev/null 2>&1; then \
	  codesign --force --sign - ${OUTFILE}; \
	  codesign --force --sign - pganalyze-collector-helper; \
	  codesign --force --sign - pganalyze-collector-setup; \
	fi

build_dist_alpine:
	# Increase stack size from Alpine's default of 80kb to 8mb (matching glibc's
	# default main-thread stack set via RLIMIT_STACK) - otherwise we see crashes
	# on very complex queries.
	go build -o ${OUTFILE} -ldflags '-extldflags "-Wl,-z,stack-size=0x800000"'
	make -C helper OUTFILE=../pganalyze-collector-helper
	make -C setup OUTFILE=../pganalyze-collector-setup

vendor:
	GO111MODULE=on go mod tidy
	GO111MODULE=on go mod vendor

test: build
	go test -race -coverprofile=coverage.out ./...

coverage: test
	go tool cover -html=coverage.out

run: build
	go run -race .

integration_test:
	make -C integration_test

packages:
	make -C packages

DOCKER_RELEASE_TAG := $(shell git describe --tags --exact-match --abbrev=0 2> /dev/null)
DOCKER_IMAGE := quay.io/pganalyze/collector
DOCKER_STAGING_IMAGE := quay.io/pganalyze/collector-staging
DOCKER_BUILDER := collector-build
DOCKER_OCI_AMD64 := pganalyze-collector-image-linux-amd64.oci.tar
DOCKER_OCI_ARM64 := pganalyze-collector-image-linux-arm64.oci.tar

# Builds one architecture as an OCI archive. The Release workflow runs these on a
# native runner per architecture and attaches the archives to the GitHub release,
# which are then published with docker_release_staging and docker_release_manifest.
define docker_build_oci
docker buildx inspect $(DOCKER_BUILDER) > /dev/null 2>&1 || docker buildx create --name $(DOCKER_BUILDER) --driver docker-container
	docker buildx build --platform linux/$(1) --builder $(DOCKER_BUILDER) \
	--provenance=true \
	--output type=oci,dest=$(2) \
	.
	docker buildx rm $(DOCKER_BUILDER)
endef

docker_image_amd64:
	$(call docker_build_oci,amd64,$(DOCKER_OCI_AMD64))

docker_image_arm64:
	$(call docker_build_oci,arm64,$(DOCKER_OCI_ARM64))

# Pushes the OCI archives downloaded from the GitHub release into the
# collector-staging repository in quay.io, which is where docker_release_manifest
# reads them from.
docker_release_staging:
	@test -n "$(DOCKER_RELEASE_TAG)" || (echo "ERROR: DOCKER_RELEASE_TAG is not set, make sure you are on a git release tag or override by setting DOCKER_RELEASE_TAG" ; exit 1)
	@test -f $(DOCKER_OCI_AMD64) || (echo "ERROR: $(DOCKER_OCI_AMD64) not found, download it with: gh release download $(DOCKER_RELEASE_TAG) -p '*.oci.tar'" ; exit 1)
	@test -f $(DOCKER_OCI_ARM64) || (echo "ERROR: $(DOCKER_OCI_ARM64) not found, download it with: gh release download $(DOCKER_RELEASE_TAG) -p '*.oci.tar'" ; exit 1)
	skopeo copy --multi-arch all oci-archive:$(DOCKER_OCI_AMD64) docker://$(DOCKER_STAGING_IMAGE):$(DOCKER_RELEASE_TAG)-amd64
	skopeo copy --multi-arch all oci-archive:$(DOCKER_OCI_ARM64) docker://$(DOCKER_STAGING_IMAGE):$(DOCKER_RELEASE_TAG)-arm64

# Publishes the release, by joining the two staged single-architecture images into
# one manifest list under the release tags.
docker_release_manifest:
	@test -n "$(DOCKER_RELEASE_TAG)" || (echo "ERROR: DOCKER_RELEASE_TAG is not set, make sure you are on a git release tag or override by setting DOCKER_RELEASE_TAG" ; exit 1)
	docker buildx imagetools create \
	-t $(DOCKER_IMAGE):$(DOCKER_RELEASE_TAG) \
	-t $(DOCKER_IMAGE):latest \
	-t $(DOCKER_IMAGE):stable \
	$(DOCKER_STAGING_IMAGE):$(DOCKER_RELEASE_TAG)-amd64 \
	$(DOCKER_STAGING_IMAGE):$(DOCKER_RELEASE_TAG)-arm64
	docker buildx imagetools inspect $(DOCKER_IMAGE):$(DOCKER_RELEASE_TAG)

output/pganalyze_collector/snapshot.pb.go: $(PROTOBUF_FILES)
ifdef PROTOC_VERSION
	mkdir -p $(PWD)/bin
	GOBIN=$(PWD)/bin go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
	protoc --go_out=. --go_opt=module=github.com/pganalyze/collector -I protobuf $(PROTOBUF_FILES)
else
	@echo '👷 Warning: protoc not found, skipping protocol buffer regeneration (to install protoc check Makefile instructions in install_protoc step)'
endif

install_protoc:
ifeq (,$(findstring $(PROTOC_VERSION_NEEDED), $(PROTOC_VERSION)))
	@echo "⚠️  protoc version needed: $(PROTOC_VERSION_NEEDED) vs $(PROTOC_VERSION) installed"
	@echo "ℹ️  Vendoring protoc $(PROTOC_VERSION_NEEDED)"

	@mkdir -p tmp
	curl --location --output tmp/$(PROTOC_FILENAME) $(PROTOC_URL)
	unzip -d protoc tmp/$(PROTOC_FILENAME)
	rm tmp/$(PROTOC_FILENAME)

	@echo "ℹ️  If this is macOS, you will need to try running the binary yourself, then go to Security & Privacy to explicitly allow it."
endif
