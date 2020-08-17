#FROM centos:7 as base 

FROM gluster/glusterfs-client AS builder

RUN yum -y update
RUN yum install -y glusterfs-api glusterfs-api-devel glusterfs-fuse gcc
RUN curl -k https://dl.google.com/go/go1.14.1.linux-amd64.tar.gz | tar xz -C /usr/local

COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh
ENV PATH=/usr/local/go/bin:$PATH

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
