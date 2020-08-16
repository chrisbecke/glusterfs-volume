FROM golang:1.13 as builder
 WORKDIR /go/src/github.com/acme/docker-volume-glusterfs
#WORKDIR /app
COPY . .
#ARG GO111MODULE=on
#RUN set -ex \
#    && apk add --no-cache --virtual .build-deps \
#    gcc libc-dev \
 #   && go get github.com/docker/go-plugins-helpers/volume \
#    && go build ./... \
#    &&
#RUN     go install --ldflags '-extldflags "-static"'dock
RUN go build -ldflags '-extldflags -static' -o docker-volume-glusterfs
 #   && apk del .build-deps

FROM oraclelinux:7-slim as final
#FROM gluster/glusterfs-client
RUN yum install -q -y oracle-gluster-release-el7
RUN yum install -y glusterfs
RUN yum install -y glusterfs-fuse
RUN yum install -y attr

RUN mkdir -p /run/docker/plugins /mnt/state /mnt/volumes
COPY --from=builder /app/docker-volume-glusterfs .
CMD ["docker-volume-glusterfs"]
