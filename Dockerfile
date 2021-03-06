FROM golang:1.8.1

RUN mkdir -p /go/src/github.com/netice9/apparatchik
WORKDIR /go/src/github.com/netice9/apparatchik
COPY . /go/src/github.com/netice9/apparatchik
RUN go install .
WORKDIR /
RUN rm -rf /go/src
CMD ["/go/bin/apparatchik"]
EXPOSE 8080
VOLUME ["/applications"]
