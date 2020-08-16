FROM gluster/glusterfs-client as base

RUN yum -y update
RUN yum install -y glusterfs-api 

FROM base AS builder

RUN yum install -y glusterfs-api-devel gcc
RUN curl -k https://dl.google.com/go/go1.14.1.linux-amd64.tar.gz | tar xz -C /usr/local

ENV PATH=/usr/local/go/bin:$PATH
WORKDIR /app
COPY . .
RUN go build

FROM base as plugin

COPY --from=builder /app/docker-volume-glusterfs /docker-volume-glusterfs

ENTRYPOINT ["docker-volume-glusterfs"]
