FROM golang:1.6
MAINTAINER Christian Parpart <christian@dawanda.com>

RUN go get github.com/tools/godep
COPY Godeps $GOPATH/src/github.com/dawanda/useronline/Godeps
COPY *.go $GOPATH/src/github.com/dawanda/useronline/

RUN cd $GOPATH/src/github.com/dawanda/useronline && \
      godep restore && \
      go install

ENTRYPOINT ["/go/bin/useronline"]
