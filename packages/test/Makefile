# Note: This requires variables that are set in the top-level packages Makefile

docker_build_and_run = docker build --no-cache --platform linux/$(1) -f Dockerfile.tmp -t pga-collector-test . && \
  docker run --platform linux/$(1) --name pga-collector-test --privileged=true -e container=docker -d pga-collector-test /sbin/init && \
  docker exec pga-collector-test bash -c "echo -n 'Waiting for systemd'; while \$$(sleep 1); do echo -n '.'; if (systemctl is-system-running 2> /dev/null || /bin/true) | grep -qE 'running|degraded'; then break; fi; done; echo ''"
docker_exec = docker exec pga-collector-test $(1)
docker_test_and_clean = docker exec pga-collector-test /root/systemd_test.sh && \
  docker kill pga-collector-test && docker rm pga-collector-test && docker rmi -f pga-collector-test

# Note: The default list excludes rockylinux9 since there is an open issue with
# /sbin/init not being present, see https://github.com/rocky-linux/sig-cloud-instance-images/issues/39
#DISTROS=centos7 rhel8 rockylinux8 rhel9 fedora34 fedora35 fedora36 amazonlinux2 ubuntu-xenial ubuntu-bionic ubuntu-focal ubuntu-jammy debian-buster debian-bullseye
DISTROS=ubuntu-xenial ubuntu-bionic ubuntu-focal ubuntu-jammy debian-buster debian-bullseye

.PHONY: all $(DISTROS)

all: $(DISTROS) clean

$(RPM_PACKAGE_X86_64):
	cp $(TMP_DIR)/$(RPM_PACKAGE_X86_64) .

$(RPM_PACKAGE_ARM64):
	cp $(TMP_DIR)/$(RPM_PACKAGE_ARM64) .

$(DEB_PACKAGE_X86_64):
	cp $(TMP_DIR)/$(DEB_PACKAGE_X86_64) .

$(DEB_PACKAGE_ARM64):
	cp $(TMP_DIR)/$(DEB_PACKAGE_ARM64) .

clean: $(RPM_PACKAGE_X86_64) $(RPM_PACKAGE_ARM64) $(DEB_PACKAGE_X86_64) $(DEB_PACKAGE_ARM64)
	rm $(RPM_PACKAGE_X86_64)
	rm $(RPM_PACKAGE_ARM64)
	rm $(DEB_PACKAGE_X86_64)
	rm $(DEB_PACKAGE_ARM64)

centos7: $(RPM_PACKAGE_X86_64) $(RPM_PACKAGE_ARM64)
	echo "FROM centos:7" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,/bin/bash -c "yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64)")
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,/bin/bash -c "yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64)")
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rhel8: $(RPM_PACKAGE_X86_64) $(RPM_PACKAGE_ARM64)
	echo "FROM redhat/ubi8:latest" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rockylinux8: $(RPM_PACKAGE_X86_64) $(RPM_PACKAGE_ARM64)
	echo "FROM rockylinux:8" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rhel9: $(RPM_PACKAGE_X86_64) $(RPM_PACKAGE_ARM64)
	echo "FROM redhat/ubi9:latest" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

rockylinux9: $(RPM_PACKAGE_X86_64) $(RPM_PACKAGE_ARM64)
	echo "FROM rockylinux:9" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

fedora34: $(RPM_PACKAGE_X86_64) $(RPM_PACKAGE_ARM64)
	echo "FROM fedora:34" > Dockerfile.tmp
	echo "RUN dnf install -y procps systemd" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

fedora35: $(RPM_PACKAGE_X86_64) $(RPM_PACKAGE_ARM64)
	echo "FROM fedora:35" > Dockerfile.tmp
	echo "RUN dnf install -y procps systemd" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

fedora36: $(RPM_PACKAGE_X86_64) $(RPM_PACKAGE_ARM64)
	echo "FROM fedora:36" > Dockerfile.tmp
	echo "RUN dnf install -y procps systemd" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

amazonlinux2: $(RPM_PACKAGE_X86_64) $(RPM_PACKAGE_ARM64)
	echo "FROM amazonlinux:2" > Dockerfile.tmp
	echo "RUN yum install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

ubuntu-xenial: $(DEB_PACKAGE_X86_64) $(DEB_PACKAGE_ARM64)
	echo "FROM ubuntu:xenial" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

ubuntu-bionic: $(DEB_PACKAGE_X86_64) $(DEB_PACKAGE_ARM64)
	echo "FROM ubuntu:bionic" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN apt-get update" >> Dockerfile.tmp
	echo "RUN apt-get install systemd-sysv -y" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

ubuntu-focal: $(DEB_PACKAGE_X86_64) $(DEB_PACKAGE_ARM64)
	echo "FROM ubuntu:focal" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN apt-get update" >> Dockerfile.tmp
	echo "RUN apt-get install systemd-sysv -y" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

ubuntu-jammy: $(DEB_PACKAGE_X86_64) $(DEB_PACKAGE_ARM64)
	echo "FROM ubuntu:jammy" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN apt-get update" >> Dockerfile.tmp
	echo "RUN apt-get install systemd-sysv -y" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

debian-jessie: $(DEB_PACKAGE_X86_64) $(DEB_PACKAGE_ARM64)
	echo "FROM debian:jessie" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

debian-buster: $(DEB_PACKAGE_X86_64) $(DEB_PACKAGE_ARM64)
	echo "FROM debian:buster" > Dockerfile.tmp
	echo "RUN apt-get update -qq && apt-get install -y -q systemd-sysv procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp

debian-bullseye: $(DEB_PACKAGE_X86_64) $(DEB_PACKAGE_ARM64)
	echo "FROM debian:bullseye" > Dockerfile.tmp
	echo "RUN apt-get update -qq && apt-get install -y -q systemd-sysv procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN rm /usr/sbin/policy-rc.d" >> Dockerfile.tmp
	$(call docker_build_and_run,amd64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_X86_64))
	$(call docker_test_and_clean)
	$(call docker_build_and_run,arm64)
	$(call docker_exec,dpkg -i /root/$(DEB_PACKAGE_ARM64))
	$(call docker_test_and_clean)
	rm Dockerfile.tmp
