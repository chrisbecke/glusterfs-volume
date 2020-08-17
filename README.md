# glusterfs-volume

This is a docker volume plugin that provisions docker volumes on glusterfs.

This plugin does not expose glusterfs volumes to docker, instead plugin instances are confugured with a single glusterfs volume, and docker volumes are allocated as sub folders.

## Quick Start

To build the plugin just run `make` and then `docker plugin ls` to verify the plugin is available.

Once the plugin is created this way, it can be connected to glusterfs storage :

```bash
docker plugin set GFS_VOLUME=gv0 GFS_SERVERS=server1,server2,server3
docker plugin enable
```

A better way is to create an instance of the plugin for each unique gluster volume it must manage:

```bash
docker plugin install --alias cloud1 gluster-volume GFS_VOLUME=cloud1 GFS_SERVERS=server1,server2,server3
```

This plugin expects the root gluster volume to be pre-created.


## GlusterFS test cluster

the root docker-compose yml includes a basic 2 server glusterfs cluster that can be interacted with. It can't do much useful as outside of docker only a single node can be interacted with.

Start Gluster

```bash
docker-compose up server1
docker-compose exec server2 peer probe server1
docker-compose exec server2 mkdir /data/brick
docker-compose exec server1 bash
gluster peer status
gluster peer probe server2
mkdir /data/brick
gluster volume create gv0 replica 2 server1:/data/brick server2:/data/brick
gluster volume start gv0
```

Note: needed to fix the glusterfs hostname in vi /var/lib/glusterd/peers/15116656-40b6-4553-8ced-21a805a1425c

## Setup of Gluster


```ini
VOLUME /run/lvm
VOLUME /sys/fs/cgroup
```

## References
* [GoDoc: package gfapi](https://godoc.org/github.com/gluster/gogfapi/gfapi)
* [GoDoc: package go-plugin-helper](https://godoc.org/github.com/docker/go-plugins-helpers/volume)
* [gfapi src](https://github.com/gluster/gogfapi)
* [go-plugin-helper src](https://github.com/docker/go-plugins-helpers)
* [config.json reference](https://docs.docker.com/engine/extend/config/)

## Logs

`docker plugin enable docker-volume-glusterfs`
```bash
2020/04/09 14:35:03 GlusterFS Volume Plugin listening on /run/docker/plugins/glusterfs.sock
2020/04/09 14:35:03 Using GlusterFS volume gv0 hosted on servers [lab717.mgsops.net lab718.mgsops.net lab719.mgsops.net]
```

`docker volume ls`
```bash
2020/04/09 14:35:15 Entering go-plugins-helpers listPath
2020/04/09 14:35:17 Entering go-plugins-helpers capabilitiesPath
```

`docker run --rm -it -v myvol:/data alpine /bin/sh`
```bash
2020/04/09 14:42:20 Entering go-plugins-helpers capabilitiesPath
2020/04/09 14:42:20 Entering go-plugins-helpers getPath
...
2020/04/09 14:42:21 Entering go-plugins-helpers capabilitiesPath
2020/04/09 14:42:21 Entering go-plugins-helpers getPath
2020/04/09 14:42:22 Entering go-plugins-helpers capabilitiesPath
...
2020/04/09 14:42:22 Entering go-plugins-helpers capabilitiesPath
2020/04/09 14:42:22 Entering go-plugins-helpers getPath
2020/04/09 14:42:23 Entering go-plugins-helpers capabilitiesPath
2020/04/09 14:42:23 Entering go-plugins-helpers mountPath
2020/04/09 14:42:23 Entered Mount &{Name:myvol ID:7c5e8ec3cc60c1526cb55d8857f8a29d38d070036ad99a9bd38e393f7ae24fdb}
2020/04/09 14:42:23 Executing &exec.Cmd{Path:\"/usr/sbin/glusterfs\", Args:[]string{\"glusterfs\", \"--volfile-server\", \"lab717.mgsops.net\", \"--volfile-server\", \"lab718.mgsops.net\", \"--volfile-server\", \"lab719.mgsops.net\", \"--volfile-id\", \"gv0\", \"--subdir-mount\", \"/myvol\", \"/mnt/volumes/myvol\"}, Env:[]string(nil), Dir:\"\", Stdin:io.Reader(nil), Stdout:io.Writer(nil), Stderr:io.Writer(nil), ExtraFiles:[]*os.File(nil), SysProcAttr:(*syscall.SysProcAttr)(nil), Process:(*os.Process)(nil), ProcessState:(*os.ProcessState)(nil), ctx:context.Context(nil), lookPathErr:error(nil), finished:false, childFiles:[]*os.File(nil), closeAfterStart:[]io.Closer(nil), closeAfterWait:[]io.Closer(nil), goroutine:[]func() error(nil), errch:(chan error)(nil), waitDone:(chan struct {})(nil)}
2020/04/09 14:42:23 Mounted registration: &{connections:1 mountpoint:/mnt/volumes/myvol ids:map[7c5e8ec3cc60c1526cb55d8857f8a29d38d070036ad99a9bd38e393f7ae24fdb:1]}
...
2020/04/09 14:42:24 Entering go-plugins-helpers capabilitiesPath
2020/04/09 14:42:24 Entering go-plugins-helpers getPath
```

`exit`
```bash
2020/04/09 14:50:32 Entering go-plugins-helpers capabilitiesPath
2020/04/09 14:50:32 Entering go-plugins-helpers getPath
2020/04/09 14:50:34 Entering go-plugins-helpers capabilitiesPath
2020/04/09 14:50:34 Entering go-plugins-helpers unmountPath
2020/04/09 14:50:34 Entered Unmount &{myvol 95d7db083ae397c15ae958bb2c35137b8ad0cd9738d3146bd85534f93e496f74}2020/04/09 14:50:34 Unmounting volume myvol with 0 clients
```

## Old Dockerfile 

Why?
```yaml
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
```