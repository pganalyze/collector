# Note: This requires variables that are set in the top-level packages Makefile

docker_build = docker build -f Dockerfile.tmp -t pga-collector-test . && rm Dockerfile.tmp
docker_run   = docker run --name pga-collector-test --privileged=true -e container=docker -d pga-collector-test /sbin/init && sleep 5
docker_exec  = docker exec pga-collector-test $(1)
docker_test  = docker exec pga-collector-test /root/$(1)_test.sh
docker_clean = docker kill pga-collector-test && docker rm pga-collector-test && docker rmi -f pga-collector-test

DISTROS=centos6 centos7 rhel8 fedora29 fedora30 ubuntu-trusty ubuntu-xenial ubuntu-bionic ubuntu-focal debian-jessie debian-stretch debian-buster

.PHONY: all $(DISTROS)

all: $(DISTROS) clean

$(RPM_SYSVINIT_PACKAGE):
	cp $(TMP_DIR)/$(RPM_SYSVINIT_PACKAGE) .

$(RPM_SYSTEMD_PACKAGE):
	cp $(TMP_DIR)/$(RPM_SYSTEMD_PACKAGE) .

$(DEB_UPSTART_PACKAGE):
	cp $(TMP_DIR)/$(DEB_UPSTART_PACKAGE) .

$(DEB_SYSTEMD_PACKAGE):
	cp $(TMP_DIR)/$(DEB_SYSTEMD_PACKAGE) .

clean:
	rm $(RPM_SYSVINIT_PACKAGE)
	rm $(RPM_SYSTEMD_PACKAGE)
	rm $(DEB_UPSTART_PACKAGE)
	rm $(DEB_SYSTEMD_PACKAGE)

centos6: $(RPM_SYSVINIT_PACKAGE)
	echo "FROM centos:6" > Dockerfile.tmp
	echo "RUN yum install -y sysvinit-tools syslog" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,service rsyslog start) # The Docker images are weird (usually this would be running)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_SYSVINIT_PACKAGE))
	$(call docker_test,sysvinit)
	$(call docker_clean)

centos7: $(RPM_SYSTEMD_PACKAGE)
	echo "FROM centos:7" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_SYSTEMD_PACKAGE))
	$(call docker_test,systemd)
	$(call docker_clean)

rhel8: $(RPM_SYSTEMD_PACKAGE)
	echo "FROM registry.access.redhat.com/ubi8/ubi:8.0-154" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,yum install -y --nogpgcheck /root/$(RPM_SYSTEMD_PACKAGE))
	$(call docker_test,systemd)
	$(call docker_clean)

fedora29: $(RPM_SYSTEMD_PACKAGE)
	echo "FROM fedora:29" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_SYSTEMD_PACKAGE))
	$(call docker_test,systemd)
	$(call docker_clean)

fedora30: $(RPM_SYSTEMD_PACKAGE)
	echo "FROM fedora:30" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,dnf install -y --nogpgcheck /root/$(RPM_SYSTEMD_PACKAGE))
	$(call docker_test,systemd)
	$(call docker_clean)

ubuntu-trusty: $(DEB_UPSTART_PACKAGE)
	echo "FROM ubuntu-upstart:trusty" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,dpkg -i /root/$(DEB_UPSTART_PACKAGE))
	$(call docker_test,upstart)
	$(call docker_clean)

ubuntu-xenial: $(DEB_SYSTEMD_PACKAGE)
	echo "FROM ubuntu:xenial" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,dpkg -i /root/$(DEB_SYSTEMD_PACKAGE))
	$(call docker_test,systemd)
	$(call docker_clean)

ubuntu-bionic: $(DEB_SYSTEMD_PACKAGE)
	echo "FROM ubuntu:bionic" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN apt-get update" >> Dockerfile.tmp
	echo "RUN apt-get install systemd-sysv -y" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,dpkg -i /root/$(DEB_SYSTEMD_PACKAGE))
	$(call docker_test,systemd)
	$(call docker_clean)

ubuntu-focal: $(DEB_SYSTEMD_PACKAGE)
	echo "FROM ubuntu:focal" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	echo "RUN apt-get update" >> Dockerfile.tmp
	echo "RUN apt-get install systemd-sysv -y" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,dpkg -i /root/$(DEB_SYSTEMD_PACKAGE))
	$(call docker_test,systemd)
	$(call docker_clean)

debian-jessie: $(DEB_SYSTEMD_PACKAGE)
	echo "FROM debian:jessie" > Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,dpkg -i /root/$(DEB_SYSTEMD_PACKAGE))
	$(call docker_test,systemd)
	$(call docker_clean)

debian-stretch: $(DEB_SYSTEMD_PACKAGE)
	echo "FROM debian:stretch" > Dockerfile.tmp
	echo "RUN apt-get update -qq && apt-get install -y -q systemd-sysv procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,dpkg -i /root/$(DEB_SYSTEMD_PACKAGE))
	$(call docker_test,systemd)
	$(call docker_clean)

debian-buster: $(DEB_SYSTEMD_PACKAGE)
	echo "FROM debian:buster" > Dockerfile.tmp
	echo "RUN apt-get update -qq && apt-get install -y -q systemd-sysv procps" >> Dockerfile.tmp
	echo "COPY . /root" >> Dockerfile.tmp
	$(call docker_build)
	$(call docker_run)
	$(call docker_exec,dpkg -i /root/$(DEB_SYSTEMD_PACKAGE))
	$(call docker_test,systemd)
	$(call docker_clean)
