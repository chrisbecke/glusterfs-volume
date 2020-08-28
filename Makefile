PLUGIN ?= gluster-volume
CONTEXT ?= default

context := $(CONTEXT)
plugin := $(PLUGIN)
go_source := main.go driver.go
volume := test
servers := lab717.mgsops.net

.PHONY: build clean image test go

all: build

bin/linux/docker-volume-glusterfs: $(go_source)
	@echo "[MAKE] Compiling glusterfs binary..."
#	GOOS=linux GOARCH=arm go build -o ./bin/linux/docker-volume-glusterfs
	docker context use default
	docker-compose --context default run builder go build -o ./bin/linux/docker-volume-glusterfs
	docker context use $(context)

image: bin/linux/docker-volume-glusterfs
	@echo "[MAKE] Building docker image for plugin..."
	@docker build -t $(plugin) .

gluster_id.txt:
	@echo "[MAKE] Creating new instance of $(plugin)"
	make image
	docker create $(plugin) > gluster_id.txt

plugin: gluster_id.txt
	@echo "[MAKE] Rebuilding plugin/rootfs..."
	mkdir -p plugin/rootfs
	docker export "$(shell cat gluster_id.txt)" | tar -x -C plugin/rootfs

plugin/config.json:	config.json
	cp $< $@

plugin/rootfs/docker-volume-glusterfs:	bin/linux/docker-volume-glusterfs
	cp $< $@

build: plugin plugin/config.json plugin/rootfs/docker-volume-glusterfs
	@echo "[MAKE] Creating docker volume plugin..."
	docker plugin disable --force $(plugin) ; true
	docker plugin rm --force $(plugin) ; true

	sudo docker plugin create $(plugin) ./plugin/

clean:
	@echo "[CLEAN] Removing container $(id)"
	docker rm -vf "$(id)" | true
	rm -f ./gluster_id.txt | true
	@echo "[CLEAN] Disabling Plugin $(plugin)"
	docker plugin disable -f $(plugin) | true
	@echo "[CLEAN] Stopping builder"
	docker-compose --context default down -v
	@echo "[CLEAN] Removing Plugin files"
	sudo rm -rf ./plugin

go: bin/linux/docker-volume-glusterfs

push: build
	docker plugin push $(plugin)

test:
	docker plugin disable $(plugin) --force | true
	docker plugin set $(plugin) GFS_VOLUME=$(volume) GFS_SERVERS=$(servers)
	docker plugin enable $(plugin)
