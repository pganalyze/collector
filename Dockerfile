FROM golang:1.13-alpine
MAINTAINER team@pganalyze.com

RUN adduser -D pganalyze pganalyze
ENV GOPATH /go
ENV HOME_DIR /home/pganalyze
ENV CODE_DIR $GOPATH/src/github.com/pganalyze/collector

COPY . $CODE_DIR
WORKDIR $CODE_DIR

# We run this all in one layer to reduce the resulting image size
RUN apk add --no-cache --virtual .build-deps make curl libc-dev gcc go git tar \
  && apk add --no-cache ca-certificates \
  && curl -o /usr/local/bin/gosu -sSL "https://github.com/tianon/gosu/releases/download/1.6/gosu-amd64" \
  && make build_dist OUTFILE=$HOME_DIR/collector \
  && rm -rf $GOPATH \
	&& apk del --purge .build-deps

RUN chmod +x /usr/local/bin/gosu
RUN chown pganalyze:pganalyze $HOME_DIR/collector

RUN mkdir /state
RUN chown pganalyze:pganalyze /state
VOLUME ["/state"]

RUN mkdir -p /usr/share/pganalyze-collector/sslrootcert/
COPY contrib/sslrootcert/rds-ca-2015-root.pem /usr/share/pganalyze-collector/sslrootcert/
COPY contrib/sslrootcert/rds-ca-2019-root.pem /usr/share/pganalyze-collector/sslrootcert/
COPY contrib/docker-entrypoint.sh $HOME_DIR
RUN chmod +x $HOME_DIR/docker-entrypoint.sh

ENTRYPOINT ["/home/pganalyze/docker-entrypoint.sh"]

CMD ["collector"]
