FROM centos:6

ENV GOPATH /go
ENV GOVERSION 1.12
ENV CODE_DIR $GOPATH/src/github.com/pganalyze/collector
ENV PATH $PATH:/usr/local/go/bin
ENV ROOT_DIR /root
ENV SOURCE_DIR /source

# Packages required for both building and packaging
RUN yum install -y -q tar make git rpmdevtools centos-release-scl

# GCC 4.6+ needed for __int128 type used by libpg_query
RUN yum install -y -q devtoolset-7-gcc

# Ruby 1.9.3+ needed for FPM
RUN yum install -y -q rh-ruby24 rh-ruby24-ruby-devel
RUN scl enable rh-ruby24 devtoolset-7 "gem install ffi -v 1.10.0" # Last release supporting Ruby 1.9
RUN scl enable rh-ruby24 devtoolset-7 "gem install fpm"

# Golang
RUN curl -o go.tar.gz -sSL "https://storage.googleapis.com/golang/go${GOVERSION}.linux-amd64.tar.gz"
RUN tar -C /usr/local -xzf go.tar.gz

# Build arguments
ARG VERSION
ARG GIT_VERSION
ENV NAME pganalyze-collector

# Build the collector
COPY . $CODE_DIR
WORKDIR $CODE_DIR
RUN git checkout ${GIT_VERSION}
RUN scl enable devtoolset-7 "make build_dist"

# Update contrib and packages directory beyond the tagged release
COPY ./contrib $CODE_DIR/contrib
COPY ./packages $CODE_DIR/packages

# Prepare the package source
RUN mkdir -p $SOURCE_DIR/usr/bin/
RUN cp $CODE_DIR/pganalyze-collector $SOURCE_DIR/usr/bin/
RUN cp $CODE_DIR/pganalyze-collector-helper $SOURCE_DIR/usr/bin/
RUN chmod +x $SOURCE_DIR/usr/bin/pganalyze-collector
RUN chmod +x $SOURCE_DIR/usr/bin/pganalyze-collector-helper
RUN mkdir -p $SOURCE_DIR/etc/
RUN cp $CODE_DIR/contrib/pganalyze-collector.conf $SOURCE_DIR/etc/pganalyze-collector.conf
RUN mkdir -p $SOURCE_DIR/etc/init.d
RUN cp $CODE_DIR/contrib/sysvinit/pganalyze-collector.init $SOURCE_DIR/etc/init.d/pganalyze-collector
RUN chmod +x $SOURCE_DIR/etc/init.d/pganalyze-collector
RUN mkdir -p $SOURCE_DIR/usr/share/pganalyze-collector/sslrootcert
RUN cp $CODE_DIR/contrib/sslrootcert/* $SOURCE_DIR/usr/share/pganalyze-collector/sslrootcert

# Build the package
WORKDIR $ROOT_DIR
RUN scl enable rh-ruby24 "/opt/rh/rh-ruby24/root/usr/local/bin/fpm \
  -n $NAME -v ${VERSION} -t rpm --rpm-os linux \
  --config-files /etc/pganalyze-collector.conf \
  --after-install $CODE_DIR/packages/src/rpm-sysvinit/post.sh \
  --before-remove $CODE_DIR/packages/src/rpm-sysvinit/preun.sh \
  -m \"<team@pganalyze.com>\" --url \"https://pganalyze.com/\" \
  --description \"pganalyze statistics collector\" \
	--vendor \"pganalyze\" --license=\"BSD\" \
  -s dir -C $SOURCE_DIR etc usr"

VOLUME ["/out"]
