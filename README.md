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

## Notes on local vs global

### scope=local
In local mode, Docker stricktly keeps its own record of each volume created and counts this volume as a usage of the plugin.

- `docker plugin rm --force` will leave these entries behind, and prevent the docker daemon from starting up cleanly.

As such, a local mode plugin, should only display volumes that have been mounted on the current node. And it should track these
to maintain parity with docker as - for display purposes - the plugins list is used. So the user has no way to delete "lost" volumes
that count against a volumes usage and prevent it being cleanly removed.

scope local plugins, when used with cloud storage, should treat volume creation as an "Attach-to-existing" verb, and removal as a "detatch" verb, as deleting volumes is the only way to remove them from the local metabase.

There is no particular way to instruct docekr to actually delete unwanted remote storage.

### scope=global

scope global plugins can be added and removed at will. docker does not track volumes on each node.
this is presumably up to the driver itself, and docker will also NOT prevent a docker volume rm reaching the driver for a volume
being used on another node.

