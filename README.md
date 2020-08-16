# docker-volume-glusterfs



## Quick Start

Start Gluster

```
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
