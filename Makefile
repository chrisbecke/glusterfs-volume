plugin = "gluster-volume"

.PHONY: build glusterfs

id := $(shell docker create docker-volume-glusterfs)

all: glusterfs

bin/linux/docker-volume-glusterfs:
		docker-compose run builder go build -o ./bin/linux/docker-volume-glusterfs

build: bin/linux/docker-volume-glusterfs
		docker build -t $(plugin) .

plugin: 
		mkdir -p plugin/rootfs
		docker export "$(id)" | tar -x -C plugin/rootfs
		docker rm -vf "$(id)"

plugin/config.json:	config.json
		cp $< $@

plugin/rootfs/docker-volume-glusterfs:	bin/linux/docker-volume-glusterfs
		cp $< $@

glusterfs: plugin plugin/config.json plugin/rootfs/docker-volume-glusterfs bin/linux/docker-volume-glusterfs
		docker plugin disable --force $(plugin) ; true
		docker plugin rm --force $(plugin) ; true
#		GOOS=linux GOARCH=arm go build -o ./plugin/rootfs/
		sudo docker plugin create $(plugin) ./plugin/

clean:
	docker plugin disable -f $(plugin)
	docker-compose down -v
	rm -rf ./plugin
