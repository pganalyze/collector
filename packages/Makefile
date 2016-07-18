NAME=pganalyze-collector
#VERSION=0.9.0
VERSION=$(shell git show -s --format=%ct.%h HEAD)

RPM_PACKAGE=$(NAME)-$(VERSION)-1.x86_64.rpm
DEB_PACKAGE=$(NAME)_$(VERSION)_amd64.deb

.PHONY: clean centos6 ubuntu-trusty push_packages_latest

all: centos6 ubuntu-trusty

clean:
	rm -f *.deb *.rpm

$(RPM_PACKAGE):
	docker build --build-arg VERSION=$(VERSION) -f Dockerfile.build.rpm-centos6 -t pga-collector-build ..
	docker run --rm -v `pwd`:/out -it pga-collector-build sh -c "cp /root/*.rpm /out"
	docker rmi pga-collector-build

$(DEB_PACKAGE):
	docker build --build-arg VERSION=$(VERSION) -f Dockerfile.build.deb-ubuntu-trusty -t pga-collector-build ..
	docker run --rm -v `pwd`:/out -it pga-collector-build sh -c "cp /root/*.deb /out"
	docker rmi pga-collector-build

centos6: $(RPM_PACKAGE)
	echo "FROM centos:6" > Dockerfile.tmp
	echo "RUN yum install -y system-config-services" >> Dockerfile.tmp
	echo "COPY "$(NAME)"-"$(VERSION)"-1.x86_64.rpm /root" >> Dockerfile.tmp
	echo "RUN yum install -y --nogpgcheck /root/"$(NAME)"-"$(VERSION)"-1.x86_64.rpm" >> Dockerfile.tmp
	docker build -f Dockerfile.tmp -t pga-collector-test .
	rm Dockerfile.tmp
	docker run --name pga-collector-test -d pga-collector-test /sbin/init
	sleep 5
	docker exec -it pga-collector-test service pganalyze-collector status | grep -q running
	docker exec -it pga-collector-test service pganalyze-collector reload
	docker exec -it pga-collector-test tail /var/log/pganalyze-collector.log | grep -q "Reloading configuration"
	docker exec -it pga-collector-test service pganalyze-collector stop
	docker kill pga-collector-test
	docker rm pga-collector-test
	docker rmi -f pga-collector-test
	echo "Test successful"

ubuntu-trusty: $(DEB_PACKAGE)
	echo "FROM ubuntu-upstart:trusty" > Dockerfile.tmp
	echo "COPY "$(NAME)"_"$(VERSION)"_amd64.deb /root" >> Dockerfile.tmp
	docker build -f Dockerfile.tmp -t pga-collector-test .
	rm Dockerfile.tmp
	docker run --name pga-collector-test -d pga-collector-test
	sleep 5
	docker exec -it pga-collector-test dpkg -i /root/$(NAME)_$(VERSION)_amd64.deb
	docker exec -it pga-collector-test status pganalyze-collector | grep -q running
	docker exec -it pga-collector-test reload pganalyze-collector
	docker exec -it pga-collector-test tail /var/log/pganalyze-collector.log | grep -q "Reloading configuration"
	docker exec -it pga-collector-test stop pganalyze-collector
	docker kill pga-collector-test
	docker rm pga-collector-test
	docker rmi -f pga-collector-test
	echo "Test successful"

push_packages_release: $(RPM_PACKAGE) $(DEB_PACKAGE)
	package_cloud push pganalyze/collector/el/6 $(NAME)-$(VERSION)-1.x86_64.rpm
	package_cloud push pganalyze/collector/ubuntu/trusty $(NAME)_$(VERSION)_amd64.deb

push_packages_latest: $(RPM_PACKAGE) $(DEB_PACKAGE)
	package_cloud push pganalyze/collector-latest/el/6 $(NAME)-$(VERSION)-1.x86_64.rpm
	package_cloud push pganalyze/collector-latest/ubuntu/trusty $(NAME)_$(VERSION)_amd64.deb
