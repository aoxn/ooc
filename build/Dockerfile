##############      builder       #############
#FROM golang:1.11.5 AS builder
#
#WORKDIR /go
#COPY . .
#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install ./...
#

############# cloud controller manager #############
FROM alpine:3.8

RUN apk add --no-cache --update ca-certificates
COPY bin/etcdctl /usr/bin/
COPY bin/ovm.amd64 /ovm
COPY bin/etcdm.sh /usr/bin/

ENTRYPOINT  ["/ovm"]
