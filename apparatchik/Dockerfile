FROM golang:1.6.2

RUN mkdir -p /go/src/github.com/netice9/apparatchik/apparatchik
WORKDIR /go/src/github.com/netice9/apparatchik/apparatchik

CMD ["/go/bin/apparatchik"]

COPY . /go/src/github.com/netice9/apparatchik/apparatchik
ENV GO15VENDOREXPERIMENT=1
ENV GOPATH=/go
RUN go install .
EXPOSE 8080
VOLUME ["/applications"]
