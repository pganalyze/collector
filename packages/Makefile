NAME=pganalyze-collector
VERSION=0.9.0rc8

.PHONY: deb-upstart

all: deb-upstart

deb-upstart:
	rm -f $(NAME)-upstart_$(VERSION)_amd64.deb
	rm -rf source/
	mkdir -p source/usr/bin/
	curl -sL https://github.com/pganalyze/collector/releases/download/v$(VERSION)/pganalyze-collector-linux-amd64 -o source/usr/bin/pganalyze-collector
	chmod +x source/usr/bin/pganalyze-collector
	mkdir -p source/etc/init
	cp ../contrib/ubuntu_upstart.conf source/etc/init/pganalyze-collector.conf
	chmod 0644 source/etc/init/pganalyze-collector.conf
	mkdir -p source/etc/init.d
	ln -s /lib/init/upstart-job source/etc/init.d/pganalyze-collector
	mkdir -p source/etc/
	cp ../contrib/sample.conf source/etc/pganalyze-collector.conf
	fpm -s dir -t deb -n $(NAME)-upstart -v $(VERSION) -C source etc usr
	rm -rf source/
	echo "FROM ubuntu-upstart:trusty" > Dockerfile.tmp
	echo "COPY "$(NAME)"-upstart_"$(VERSION)"_amd64.deb /root" >> Dockerfile.tmp
	echo "RUN dpkg -i /root/"$(NAME)"-upstart_"$(VERSION)"_amd64.deb" >> Dockerfile.tmp
	docker build -f Dockerfile.tmp -t pga-collector-test-ubuntu-trusty .
	rm Dockerfile.tmp
	docker run --name pga-collector-test -d pga-collector-test-ubuntu-trusty
	docker exec -it pga-collector-test status pganalyze-collector | grep -q running
	docker exec -it pga-collector-test reload pganalyze-collector
	docker exec -it pga-collector-test tail /var/log/pganalyze-collector.log | grep -q "Reloading configuration"
	docker exec -it pga-collector-test stop pganalyze-collector
	docker kill pga-collector-test
	docker rm pga-collector-test
	echo "Test successful"
