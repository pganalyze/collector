FROM debian:jessie

# Build arguments
ARG VERSION
ENV NAME pganalyze-collector

ENV DEB_DIR /deb
RUN mkdir -p $DEB_DIR
RUN mkdir -p $DEB_DIR/upstart
RUN mkdir -p $DEB_DIR/systemd

RUN apt-get update -qq && apt-get install -y -q reprepro

COPY sync_deb.sh /root
COPY deb.distributions /root
COPY ${NAME}_${VERSION}_upstart_amd64.deb $DEB_DIR/upstart/${NAME}_${VERSION}_amd64.deb
COPY ${NAME}_${VERSION}_systemd_amd64.deb $DEB_DIR/systemd/${NAME}_${VERSION}_amd64.deb

VOLUME ["/repo"]
