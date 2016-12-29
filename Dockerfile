FROM golang:1.7.3

RUN mkdir -p /go/src/github.com/netice9/apparatchik
WORKDIR /go/src/github.com/netice9/apparatchik
COPY . /go/src/github.com/netice9/apparatchik
# ENV GOPATH=/go
RUN go install .

CMD ["/go/bin/apparatchik"]
EXPOSE 8080
VOLUME ["/applications"]
