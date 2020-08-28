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

## Publish to a remote repository

docker-compose builder operations are passed an explicit --context default flag so that, if you wish, you can set a remote
docker agent as the build target to deploy directly to a swarm server (as an example)

```bash
make build plugin=registry.unreal.mgsops.net/gluster-volume
```

## GlusterFS test cluster

the root docker-compose yml includes a basic 2 server glusterfs cluster that can be interacted with. It can't do much useful as outside of docker only a single node can be interacted with. Unfortunately it doesnt help to provide storage
for the plugin right now.

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

## Debugging

### Plugin Logs

Macos: ~/Library/Containers/com.docker.docker/Data/log/vm/dockerd.log
RHEL: journalctl -xfu docker.service

### Attach to plugins

```bash
runc --root /var/run/docker/plugins/runtime-root/plugins.moby/ list
runc --root /var/run/docker/plugins/runtime-root/plugins.moby/ exec -t 5693b036ce049834b29fa7f00547dc6f89e626c5814987cb805f905dba5d5358 /bin/sh
```

## Implementation Notes

### docker plugin enable

Activates the plugin

### docker volume ls

* listPath
* capabilities Path

### docker run -v ...

* capabilitiesPath
* getPath
* mountPath

On Container Exit:

* capabilitesPath
* getPath
* unmountPath

### docker volume create -d 

* capabilitiesPath
* getPath
if getPath fails
* createPath

### docker volume rm

If containers are using the volume
* capabilitiesPath
* getPath

If containers are not using the volume
* capabilitiesPath
* getPath
* removePath