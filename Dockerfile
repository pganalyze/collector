FROM golang:1.17-alpine
MAINTAINER team@pganalyze.com

RUN adduser -D pganalyze pganalyze
ENV GOPATH /go
ENV HOME_DIR /home/pganalyze
ENV CODE_DIR $GOPATH/src/github.com/pganalyze/collector

COPY . $CODE_DIR
WORKDIR $CODE_DIR

# We run this all in one layer to reduce the resulting image size
RUN apk add --no-cache --virtual .build-deps make curl libc-dev gcc go git tar \
  && apk add --no-cache ca-certificates setpriv \
  && make build_dist_alpine OUTFILE=$HOME_DIR/collector \
  && rm -rf $GOPATH \
	&& apk del --purge .build-deps

RUN chown pganalyze:pganalyze $HOME_DIR/collector

RUN mkdir /state
RUN chown pganalyze:pganalyze /state
VOLUME ["/state"]

RUN mkdir -p /usr/share/pganalyze-collector/sslrootcert/
COPY contrib/sslrootcert/rds-ca-2015-root.pem /usr/share/pganalyze-collector/sslrootcert/
COPY contrib/sslrootcert/rds-ca-2019-root.pem /usr/share/pganalyze-collector/sslrootcert/
COPY contrib/docker-entrypoint.sh $HOME_DIR
RUN chmod +x $HOME_DIR/docker-entrypoint.sh

USER pganalyze

ENTRYPOINT ["/home/pganalyze/docker-entrypoint.sh"]

CMD ["collector"]
