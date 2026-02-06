# Note: This requires variables that are set in the top-level packages Makefile

DOCKER_CMD=docker

docker_build_and_run = $(DOCKER_CMD) build --no-cache --platform linux/$(1) -f Dockerfile.tmp -t pga-collector-test . && \
  $(DOCKER_CMD) run --platform linux/$(1) --name pga-collector-test --privileged=true -e container=docker -d pga-collector-test /sbin/init && \
  $(DOCKER_CMD) exec pga-collector-test bash -c "echo -n 'Waiting for systemd'; while \$$(sleep 1); do echo -n '.'; if (systemctl is-system-running 2> /dev/null || /bin/true) | grep -qE 'running|degraded'; then break; fi; done; echo ''"
docker_exec = $(DOCKER_CMD) exec pga-collector-test $(1)
docker_test_and_clean = $(DOCKER_CMD) exec pga-collector-test /root/systemd_test.sh && \
  $(DOCKER_CMD) kill pga-collector-test && $(DOCKER_CMD) rm pga-collector-test && $(DOCKER_CMD) rmi -f pga-collector-test

# Note: The default list excludes rockylinux9 since there is an open issue with
# /sbin/init not being present, see https://github.com/rocky-linux/sig-cloud-instance-images/issues/39
# Note: The default list also excludes centos7 since it's EOL, we will drop the support near future
DISTROS_X86_64=rhel8_x86_64 rockylinux8_x86_64 rhel9_x86_64 fedora36_x86_64 fedora37_x86_64 amazonlinux2_x86_64 amazonlinux2023_x86_64 ubuntu-focal_x86_64 ubuntu-jammy_x86_64 ubuntu-noble_x86_64 debian-bullseye_x86_64 debian-bookworm_x86_64 debian-trixie_x86_64
DISTROS_ARM64=rhel8_arm64 rockylinux8_arm64 rhel9_arm64 fedora36_arm64 fedora37_arm64 amazonlinux2_arm64 amazonlinux2023_arm64 ubuntu-focal_arm64 ubuntu-jammy_arm64 ubuntu-noble_arm64 debian-bullseye_arm64 debian-bookworm_arm64 debian-trixie_arm64

CGROUPS_V1 := $(shell test `stat -fc %T /sys/fs/cgroup/` = tmpfs && echo '1' 1>&2 2> /dev/null)

.PHONY: all $(DISTROS_X86_64) $(DISTROS_ARM64)

all: $(DISTROS_X86_64) $(DISTROS_ARM64) clean

x86_64: $(DISTROS_X86_64) clean_x86_64

arm64: $(DISTROS_ARM64) clean_arm64

$(RPM_PACKAGE_X86_64):
	cp $(TMP_DIR)/$(RPM_PACKAGE_X86_64) .

$(RPM_PACKAGE_ARM64):
	cp $(TMP_DIR)/$(RPM_PACKAGE_ARM64) .

$(DEB_PACKAGE_X86_64):
	cp $(TMP_DIR)/$(DEB_PACKAGE_X86_64) .

$(DEB_PACKAGE_ARM64):
	cp $(TMP_DIR)/$(DEB_PACKAGE_ARM64) .

clean_x86_64: $(RPM_PACKAGE_X86_64) $(DEB_PACKAGE_X86_64)
	rm $(RPM_PACKAGE_X86_64)
	rm $(DEB_PACKAGE_X86_64)

clean_arm64: $(RPM_PACKAGE_ARM64) $(DEB_PACKAGE_ARM64)
	rm $(RPM_PACKAGE_ARM64)
	rm $(DEB_PACKAGE_ARM64)

clean: clean_x86_64 clean_arm64

centos7_x86_64: $(RPM_PACKAGE_X86_64)
	echo "FROM centos:7" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,/bin/bash -c "yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64)")
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

centos7_arm64: $(RPM_PACKAGE_ARM64)
	echo "FROM centos:7" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,/bin/bash -c "yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64)")
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rhel8_x86_64: $(RPM_PACKAGE_X86_64)
	echo "FROM redhat/ubi8:latest" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rhel8_arm64: $(RPM_PACKAGE_ARM64)
	echo "FROM redhat/ubi8:latest" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rockylinux8_x86_64: $(RPM_PACKAGE_X86_64)
	echo "FROM rockylinux:8" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rockylinux8_arm64: $(RPM_PACKAGE_ARM64)
	echo "FROM rockylinux:8" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rhel9_x86_64: $(RPM_PACKAGE_X86_64)
	echo "FROM redhat/ubi9:latest" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rhel9_arm64: $(RPM_PACKAGE_ARM64)
	echo "FROM redhat/ubi9:latest" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rockylinux9_x86_64: $(RPM_PACKAGE_X86_64)
	echo "FROM rockylinux:9" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rockylinux9_arm64: $(RPM_PACKAGE_ARM64)
	echo "FROM rockylinux:9" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

fedora36_x86_64: $(RPM_PACKAGE_X86_64)
	echo "FROM fedora:36" > Dockerfile.tmp
	echo "RUN dnf install -y procps systemd" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

fedora36_arm64: $(RPM_PACKAGE_ARM64)
	echo "FROM fedora:36" > Dockerfile.tmp
	echo "RUN dnf install -y procps systemd" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

fedora37_x86_64: $(RPM_PACKAGE_X86_64)
	echo "FROM fedora:37" > Dockerfile.tmp
	echo "RUN dnf install -y procps systemd" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

fedora37_arm64: $(RPM_PACKAGE_ARM64)
	echo "FROM fedora:37" > Dockerfile.tmp
	echo "RUN dnf install -y procps systemd" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

amazonlinux2_x86_64: $(RPM_PACKAGE_X86_64)
	echo "FROM amazonlinux:2" > Dockerfile.tmp
	echo "RUN yum install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
ifeq ($(CGROUPS_V1),1)
	$(call docker_build_and_run,amd64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
else
	echo "Skipping Amazon Linux 2 test since host system uses cgroup v2"
endif
	rm Dockerfile.tmp

amazonlinux2_arm64: $(RPM_PACKAGE_ARM64)
	echo "FROM amazonlinux:2" > Dockerfile.tmp
	echo "RUN yum install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
ifeq ($(CGROUPS_V1),1)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
else
	echo "Skipping Amazon Linux 2 test since host system uses cgroup v2"
endif
	rm Dockerfile.tmp

amazonlinux2023_x86_64: $(RPM_PACKAGE_X86_64)
	echo "FROM amazonlinux:2023" > Dockerfile.tmp
	echo "RUN yum install -y procps systemd" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

amazonlinux2023_arm64: $(RPM_PACKAGE_ARM64)
	echo "FROM amazonlinux:2023" > Dockerfile.tmp
	echo "RUN yum install -y procps systemd" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

ubuntu-focal_x86_64: $(DEB_PACKAGE_X86_64)
	echo "FROM ubuntu:focal" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN apt-get update" >> Dockerfile.tmp
	echo "RUN apt-get install systemd-sysv -y" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

ubuntu-focal_arm64: $(DEB_PACKAGE_ARM64)
	echo "FROM ubuntu:focal" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN apt-get update" >> Dockerfile.tmp
	echo "RUN apt-get install systemd-sysv -y" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

ubuntu-jammy_x86_64: $(DEB_PACKAGE_X86_64)
	echo "FROM ubuntu:jammy" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN apt-get update" >> Dockerfile.tmp
	echo "RUN apt-get install systemd-sysv -y" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

ubuntu-jammy_arm64: $(DEB_PACKAGE_ARM64)
	echo "FROM ubuntu:jammy" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN apt-get update" >> Dockerfile.tmp
	echo "RUN apt-get install systemd-sysv -y" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

ubuntu-noble_x86_64: $(DEB_PACKAGE_X86_64)
	echo "FROM ubuntu:noble" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN apt-get update" >> Dockerfile.tmp
	echo "RUN apt-get install systemd-sysv -y" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

ubuntu-noble_arm64: $(DEB_PACKAGE_ARM64)
	echo "FROM ubuntu:noble" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN apt-get update" >> Dockerfile.tmp
	echo "RUN apt-get install systemd-sysv -y" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

debian-bullseye_x86_64: $(DEB_PACKAGE_X86_64)
	echo "FROM debian:bullseye" > Dockerfile.tmp
	echo "RUN apt-get update -qq && apt-get install -y -q systemd-sysv procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

debian-bullseye_arm64: $(DEB_PACKAGE_ARM64)
	echo "FROM debian:bullseye" > Dockerfile.tmp
	echo "RUN apt-get update -qq && apt-get install -y -q systemd-sysv procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

debian-bookworm_x86_64: $(DEB_PACKAGE_X86_64)
	echo "FROM debian:bookworm" > Dockerfile.tmp
	echo "RUN apt-get update -qq && apt-get install -y -q systemd-sysv procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

debian-bookworm_arm64: $(DEB_PACKAGE_ARM64)
	echo "FROM debian:bookworm" > Dockerfile.tmp
	echo "RUN apt-get update -qq && apt-get install -y -q systemd-sysv procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

debian-trixie_x86_64: $(DEB_PACKAGE_X86_64)
	echo "FROM debian:trixie" > Dockerfile.tmp
	echo "RUN apt-get update -qq && apt-get install -y -q systemd-sysv procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

debian-trixie_arm64: $(DEB_PACKAGE_ARM64)
	echo "FROM debian:trixie" > Dockerfile.tmp
	echo "RUN apt-get update -qq && apt-get install -y -q systemd-sysv procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp
