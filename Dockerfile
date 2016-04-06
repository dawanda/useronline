FROM golang:1.6
MAINTAINER Christian Parpart <christian@dawanda.com>

COPY *.go $GOPATH/src/github.com/dawanda/useronline/
RUN cd $GOPATH/src/github.com/dawanda/useronline && go install

ENTRYPOINT ["/go/bin/useronline"]
