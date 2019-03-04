FROM postgres:10

ENV GOVERSION 1.12
ENV CODE_DIR /collector
ENV PATH $PATH:/usr/local/go/bin

# Packages required for both building and packaging
RUN apt-get update -qq && apt-get install -y -q build-essential git curl

# Golang
RUN curl -o go.tar.gz -sSL "https://storage.googleapis.com/golang/go${GOVERSION}.linux-amd64.tar.gz"
RUN tar -C /usr/local -xzf go.tar.gz

# Build the collector
COPY . $CODE_DIR
WORKDIR $CODE_DIR
RUN make build_dist

# Make sure collector state can be saved
RUN mkdir /var/lib/pganalyze-collector/

RUN cp $CODE_DIR/pganalyze-collector /usr/bin/
RUN cp $CODE_DIR/pganalyze-collector-helper /usr/bin/
