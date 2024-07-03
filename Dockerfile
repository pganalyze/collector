FROM golang:1.21-alpine AS base
LABEL org.opencontainers.image.authors="team@pganalyze.com"

ENV GOPATH=/go
ENV HOME_DIR=/home/pganalyze
ENV CODE_DIR=$GOPATH/src/github.com/pganalyze/collector

RUN apk add --no-cache --virtual .build-deps make curl libc-dev gcc git tar \
  && mkdir -p $HOME_DIR

COPY . $CODE_DIR
WORKDIR $CODE_DIR

RUN make build_dist_alpine OUTFILE=$HOME_DIR/collector

RUN mkdir -p /usr/share/pganalyze-collector/sslrootcert/
COPY contrib/sslrootcert/rds-ca-global.pem /usr/share/pganalyze-collector/sslrootcert/
COPY contrib/docker-entrypoint.sh $HOME_DIR
RUN chmod +x $HOME_DIR/docker-entrypoint.sh

FROM alpine:3.18 AS slim

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D pganalyze pganalyze \
  && mkdir /state  \
  && chown pganalyze:pganalyze /state \
  && mkdir /config \
  && chown pganalyze:pganalyze /config

COPY --from=base --chown=pganalyze:pganalyze /home/pganalyze/docker-entrypoint.sh /home/pganalyze/collector /home/pganalyze/
COPY --from=base /usr/share/pganalyze-collector/sslrootcert/ /usr/share/pganalyze-collector/sslrootcert/

VOLUME ["/state", "/config"]

USER pganalyze

ENTRYPOINT ["/home/pganalyze/docker-entrypoint.sh"]

CMD ["collector"]
