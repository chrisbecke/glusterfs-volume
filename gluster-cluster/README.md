# GlusterFS compose cluster

## Quick Start

`docker-compose up -d server1 server2` brings up a test cluster

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