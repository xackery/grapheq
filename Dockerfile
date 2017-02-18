FROM golang:1.6.2
MAINTAINER Xackery <xackery@gmail.com>

ENV GOPATH /go
ENV USER root

# pre-install known dependencies before the source, so we don't redownload them whenever the source changes
RUN go get github.com/go-sql-driver/mysql \
	&& go get github.com/jmoiron/sqlx \
	&& go get github.com/prometheus/client_golang/prometheus \
	&& go get github.com/prometheus/client_golang/prometheus/promhttp \
	&& go get github.com/xackery/eqemuconfig 

COPY . /go/src/github.com/xackery/grapheq

RUN cd /go/src/github.com/xackery/grapheq \
	&& go get -d -v \
	&& go install \
	&& go test github.com/xackery/grapheq...
