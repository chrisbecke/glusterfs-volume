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

FROM alpine as tini
# Add Tini
ENV TINI_VERSION v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini /tini
RUN chmod +x /tini

FROM base as plugin

COPY --from=tini /tini /tini
COPY --from=builder /app/docker-volume-glusterfs /docker-volume-glusterfs

ENTRYPOINT ["/tini", "--"]
CMD ["docker-volume-glusterfs"]
