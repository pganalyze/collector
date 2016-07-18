NAME=pganalyze-collector
VERSION=0.9.0
#VERSION=$(shell git show -s --format=%ct.%h HEAD)

RPM_PACKAGE_OUT=$(NAME)-$(VERSION)-1.x86_64.rpm
RPM_SYSVINIT_PACKAGE=$(NAME)-$(VERSION)-1_sysvinit.x86_64.rpm
RPM_SYSTEMD_PACKAGE=$(NAME)-$(VERSION)-1_systemd.x86_64.rpm
DEB_PACKAGE_OUT=$(NAME)_$(VERSION)_amd64.deb
DEB_UPSTART_PACKAGE=$(NAME)_$(VERSION)_upstart_amd64.deb
DEB_SYSTEMD_PACKAGE=$(NAME)_$(VERSION)_systemd_amd64.deb

DISTROS=centos6 centos7 fedora24 ubuntu-trusty ubuntu-xenial debian-jessie debian-stretch

.PHONY: clean $(DISTROS)

all: $(DISTROS)

clean:
	rm -f *.deb *.rpm

$(RPM_SYSVINIT_PACKAGE):
	docker build --build-arg VERSION=$(VERSION) -f Dockerfile.build.rpm-sysvinit -t pga-collector-build ..
	docker run --rm -v `pwd`:/out -it pga-collector-build sh -c "cp /root/*.rpm /out"
	mv $(RPM_PACKAGE_OUT) $(RPM_SYSVINIT_PACKAGE)
	docker rmi pga-collector-build

$(RPM_SYSTEMD_PACKAGE):
	docker build --build-arg VERSION=$(VERSION) -f Dockerfile.build.rpm-systemd -t pga-collector-build ..
	docker run --rm -v `pwd`:/out -it pga-collector-build sh -c "cp /root/*.rpm /out"
	mv $(RPM_PACKAGE_OUT) $(RPM_SYSTEMD_PACKAGE)
	docker rmi pga-collector-build

$(DEB_UPSTART_PACKAGE):
	docker build --build-arg VERSION=$(VERSION) -f Dockerfile.build.deb-upstart -t pga-collector-build ..
	docker run --rm -v `pwd`:/out -it pga-collector-build sh -c "cp /root/*.deb /out"
	mv $(DEB_PACKAGE_OUT) $(DEB_UPSTART_PACKAGE)
	docker rmi pga-collector-build

$(DEB_SYSTEMD_PACKAGE):
	docker build --build-arg VERSION=$(VERSION) -f Dockerfile.build.deb-systemd -t pga-collector-build ..
	docker run --rm -v `pwd`:/out -it pga-collector-build sh -c "cp /root/*.deb /out"
	mv $(DEB_PACKAGE_OUT) $(DEB_SYSTEMD_PACKAGE)
	docker rmi pga-collector-build

centos6: $(RPM_SYSVINIT_PACKAGE)
	echo "FROM centos:6" > Dockerfile.tmp
	echo "RUN yum install -y sysvinit-tools syslog" >> Dockerfile.tmp
	echo "COPY "$(RPM_SYSVINIT_PACKAGE)" /root" >> Dockerfile.tmp
	docker build -f Dockerfile.tmp -t pga-collector-test .
	rm Dockerfile.tmp
	docker run --name pga-collector-test -d pga-collector-test /sbin/init
	sleep 5
	docker exec pga-collector-test service rsyslog start # The Docker images are weird (usually this would be running)
	docker exec pga-collector-test yum install -y --nogpgcheck /root/$(RPM_SYSVINIT_PACKAGE)
	docker exec pga-collector-test service pganalyze-collector status | grep -q running
	docker exec pga-collector-test ps u -U pganalyze | grep -q pganalyze-collector
	docker exec pga-collector-test service pganalyze-collector reload
	docker exec pga-collector-test tail /var/log/messages | grep -q "Reloading configuration"
	docker exec pga-collector-test service pganalyze-collector stop
	docker kill pga-collector-test
	docker rm pga-collector-test
	docker rmi -f pga-collector-test
	echo "Test successful"

centos7: $(RPM_SYSTEMD_PACKAGE)
	echo "FROM centos:7" > Dockerfile.tmp
	echo "COPY "$(RPM_SYSTEMD_PACKAGE)" /root" >> Dockerfile.tmp
	docker build -f Dockerfile.tmp -t pga-collector-test .
	rm Dockerfile.tmp
	docker run --name pga-collector-test --privileged=true -e container=docker -d pga-collector-test /sbin/init
	sleep 5
	docker exec pga-collector-test yum install -y --nogpgcheck /root/$(RPM_SYSTEMD_PACKAGE)
	docker exec pga-collector-test systemctl status pganalyze-collector | grep -q running
	docker exec pga-collector-test ps u -U pganalyze | grep -q pganalyze-collector
	docker exec pga-collector-test systemctl reload pganalyze-collector
	docker exec pga-collector-test journalctl -u pganalyze-collector | grep -q "Reloading configuration"
	docker exec pga-collector-test systemctl stop pganalyze-collector
	docker kill pga-collector-test
	docker rm pga-collector-test
	docker rmi -f pga-collector-test
	echo "Test successful"

fedora24: $(RPM_SYSTEMD_PACKAGE)
	echo "FROM fedora:24" > Dockerfile.tmp
	echo "RUN dnf install -y procps" >> Dockerfile.tmp
	echo "COPY "$(RPM_SYSTEMD_PACKAGE)" /root" >> Dockerfile.tmp
	docker build -f Dockerfile.tmp -t pga-collector-test .
	rm Dockerfile.tmp
	docker run --name pga-collector-test --privileged=true -e container=docker -d pga-collector-test /sbin/init
	sleep 5
	docker exec pga-collector-test dnf install -y --nogpgcheck /root/$(RPM_SYSTEMD_PACKAGE)
	docker exec pga-collector-test systemctl status pganalyze-collector | grep -q running
	docker exec pga-collector-test ps u -U pganalyze | grep -q pganalyze-collector
	docker exec pga-collector-test systemctl reload pganalyze-collector
	docker exec pga-collector-test journalctl -u pganalyze-collector | grep -q "Reloading configuration"
	docker exec pga-collector-test systemctl stop pganalyze-collector
	docker kill pga-collector-test
	docker rm pga-collector-test
	docker rmi -f pga-collector-test
	echo "Test successful"

ubuntu-trusty: $(DEB_UPSTART_PACKAGE)
	echo "FROM ubuntu-upstart:trusty" > Dockerfile.tmp
	echo "COPY "$(DEB_UPSTART_PACKAGE)" /root" >> Dockerfile.tmp
	docker build -f Dockerfile.tmp -t pga-collector-test .
	rm Dockerfile.tmp
	docker run --name pga-collector-test -d pga-collector-test
	sleep 5
	docker exec pga-collector-test rsyslogd # The Docker images are weird (usually this would be running)
	docker exec pga-collector-test dpkg -i /root/$(DEB_UPSTART_PACKAGE)
	docker exec pga-collector-test status pganalyze-collector | grep -q running
	docker exec pga-collector-test ps u -U pganalyze | grep -q pganalyze-collector
	docker exec pga-collector-test reload pganalyze-collector
	docker exec pga-collector-test tail /var/log/syslog | grep -q "Reloading configuration"
	docker exec pga-collector-test stop pganalyze-collector
	docker kill pga-collector-test
	docker rm pga-collector-test
	docker rmi -f pga-collector-test
	echo "Test successful"

ubuntu-xenial: $(DEB_SYSTEMD_PACKAGE)
	echo "FROM ubuntu:xenial" > Dockerfile.tmp
	echo "COPY "$(DEB_SYSTEMD_PACKAGE)" /root" >> Dockerfile.tmp
	docker build -f Dockerfile.tmp -t pga-collector-test .
	rm Dockerfile.tmp
	docker run --name pga-collector-test --privileged=true -e container=docker -d pga-collector-test /sbin/init
	sleep 5
	docker exec pga-collector-test dpkg -i /root/$(DEB_SYSTEMD_PACKAGE)
	docker exec pga-collector-test systemctl status pganalyze-collector | grep -q running
	docker exec pga-collector-test ps u -U pganalyze | grep -q pganalyze-collector
	docker exec pga-collector-test systemctl reload pganalyze-collector
	docker exec pga-collector-test journalctl -u pganalyze-collector | grep -q "Reloading configuration"
	docker exec pga-collector-test systemctl stop pganalyze-collector
	docker kill pga-collector-test
	docker rm pga-collector-test
	docker rmi -f pga-collector-test
	echo "Test successful"

debian-jessie: $(DEB_SYSTEMD_PACKAGE)
	echo "FROM debian:jessie" > Dockerfile.tmp
	echo "COPY "$(DEB_SYSTEMD_PACKAGE)" /root" >> Dockerfile.tmp
	docker build -f Dockerfile.tmp -t pga-collector-test .
	rm Dockerfile.tmp
	docker run --name pga-collector-test --privileged=true -e container=docker -d pga-collector-test /sbin/init
	sleep 5
	docker exec pga-collector-test dpkg -i /root/$(DEB_SYSTEMD_PACKAGE)
	docker exec pga-collector-test systemctl status pganalyze-collector | grep -q running
	docker exec pga-collector-test ps u -U pganalyze | grep -q pganalyze-collector
	docker exec pga-collector-test systemctl reload pganalyze-collector
	docker exec pga-collector-test journalctl -u pganalyze-collector | grep -q "Reloading configuration"
	docker exec pga-collector-test systemctl stop pganalyze-collector
	docker kill pga-collector-test
	docker rm pga-collector-test
	docker rmi -f pga-collector-test
	echo "Test successful"

debian-stretch: $(DEB_SYSTEMD_PACKAGE)
	echo "FROM debian:stretch" > Dockerfile.tmp
	echo "COPY "$(DEB_SYSTEMD_PACKAGE)" /root" >> Dockerfile.tmp
	echo "RUN apt-get update -qq && apt-get install -y -q systemd-sysv procps" >> Dockerfile.tmp
	docker build -f Dockerfile.tmp -t pga-collector-test .
	rm Dockerfile.tmp
	docker run --name pga-collector-test --privileged=true -e container=docker -d pga-collector-test /sbin/init
	sleep 5
	docker exec pga-collector-test dpkg -i /root/$(DEB_SYSTEMD_PACKAGE)
	docker exec pga-collector-test systemctl status pganalyze-collector | grep -q running
	docker exec pga-collector-test ps u -U pganalyze | grep -q pganalyze-collector
	docker exec pga-collector-test systemctl reload pganalyze-collector
	docker exec pga-collector-test journalctl -u pganalyze-collector | grep -q "Reloading configuration"
	docker exec pga-collector-test systemctl stop pganalyze-collector
	docker kill pga-collector-test
	docker rm pga-collector-test
	docker rmi -f pga-collector-test
	echo "Test successful"

push_packages_release: $(DISTROS)
	package_cloud push pganalyze/collector/el/6 $(RPM_SYSVINIT_PACKAGE)
	package_cloud push pganalyze/collector/el/7 $(RPM_SYSTEMD_PACKAGE)
	package_cloud push pganalyze/collector/fedora/24 $(RPM_SYSTEMD_PACKAGE)
	package_cloud push pganalyze/collector/ubuntu/trusty $(DEB_UPSTART_PACKAGE)
	package_cloud push pganalyze/collector/ubuntu/xenial $(DEB_SYSTEMD_PACKAGE)
	package_cloud push pganalyze/collector/debian/jessie $(DEB_SYSTEMD_PACKAGE)
	package_cloud push pganalyze/collector/debian/stretch $(DEB_SYSTEMD_PACKAGE)

push_packages_latest: $(DISTROS)
	package_cloud push pganalyze/collector-latest/el/6 $(RPM_SYSVINIT_PACKAGE)
	package_cloud push pganalyze/collector-latest/el/7 $(RPM_SYSTEMD_PACKAGE)
	package_cloud push pganalyze/collector-latest/fedora/24 $(RPM_SYSTEMD_PACKAGE)
	package_cloud push pganalyze/collector-latest/ubuntu/trusty $(DEB_UPSTART_PACKAGE)
	package_cloud push pganalyze/collector-latest/ubuntu/xenial $(DEB_SYSTEMD_PACKAGE)
	package_cloud push pganalyze/collector-latest/debian/jessie $(DEB_SYSTEMD_PACKAGE)
	package_cloud push pganalyze/collector-latest/debian/stretch $(DEB_SYSTEMD_PACKAGE)
