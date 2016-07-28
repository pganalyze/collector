FROM centos:7

# Build arguments
ARG VERSION
ENV NAME pganalyze-collector

ENV RPM_DIR /rpm
RUN mkdir -p $RPM_DIR
RUN mkdir -p $RPM_DIR/sysvinit
RUN mkdir -p $RPM_DIR/systemd

RUN yum install -y -q rpm-sign createrepo

RUN echo "%_gpg_name team@pganalyze.com" > /root/.rpmmacros

COPY sync_rpm.sh /root
COPY $NAME-$VERSION-1_sysvinit.x86_64.rpm $RPM_DIR/sysvinit/$NAME-$VERSION-1.x86_64.rpm
COPY $NAME-$VERSION-1_systemd.x86_64.rpm $RPM_DIR/systemd/$NAME-$VERSION-1.x86_64.rpm

VOLUME ["/repo"]
