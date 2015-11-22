FROM golang:1.5.1

RUN mkdir -p /go/src/app
WORKDIR /go/src/app

CMD ["go-wrapper", "run"]

COPY . /go/src/app
ENV GO15VENDOREXPERIMENT=1
RUN go-wrapper install
EXPOSE 8080
VOLUME ["/applications"]
