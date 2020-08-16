.PHONY: docker rootfs glusterfs

id := $(shell docker create docker-volume-glusterfs)

build:
		docker-compose run builder go build -o ./bin/linux/docker-volume-glusterfs
#		GOOS=linux GOARCH=amd64 go build -o ./bin/linux/docker-volume-glusterfs

docker: build
		docker build -t docker-volume-glusterfs .

plugin: 
		mkdir -p plugin/rootfs
		docker export "$(id)" | tar -x -C plugin/rootfs
		docker rm -vf "$(id)"

# How do I get these files to be explicitly in scope?
build/linux/docker-volume-glusterfs: build
		cp $< $@
plugin/config.json:	config.json
		cp $< $@
plugin/rootfs/docker-volume-glusterfs:	build/linux/docker-volume-glusterfs
		cp $< $@

glusterfs: plugin docker
		cp ./config.json ./plugin/
		cp bin/linux/docker-volume-glusterfs plugin/rootfs/docker-volume-glusterfs
		docker plugin disable --force docker-volume-glusterfs ; true
		docker plugin rm --force docker-volume-glusterfs ; true
#		GOOS=linux GOARCH=arm go build -o ./plugin/rootfs/
		sudo docker plugin create docker-volume-glusterfs ./plugin/

all: glusterfs

clean:
	docker-compose down -v
