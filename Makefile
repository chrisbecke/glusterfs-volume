plugin = $(notdir $(CURDIR))
context = default
go_source := main.go driver.go gfs.go
volume := test
# gv0
servers := lab717.mgsops.net,lab718.mgsops.net,lab719.mgsops.net
version = 4

.PHONY: build clean image test go

all: build

bin/linux/docker-volume-glusterfs: $(go_source)
	@echo "[MAKE] Compiling glusterfs binary..."
#	GOOS=linux GOARCH=arm go build -o ./bin/linux/docker-volume-glusterfs
	@DOCKER_BUILDKIT=1 \
	COMPOSE_DOCKER_CLI_BUILD=1 \
	docker-compose --context default run builder go build -o ./bin/linux/docker-volume-glusterfs

image: bin/linux/docker-volume-glusterfs
	@echo "[MAKE] Building docker image for plugin..."
	@DOCKER_BUILDKIT=1 \
	docker build -t $(plugin) .

gluster_id.txt:
	@echo "[MAKE] Creating new instance of $(plugin)"
	make image
	docker create $(plugin) > gluster_id.txt

plugin: gluster_id.txt
	@echo "[MAKE] Rebuilding plugin/rootfs..."
	@mkdir -p plugin/rootfs
	@docker -c default export "$(shell cat gluster_id.txt)" | tar -x -C plugin/rootfs

plugin/config.json:	config.json
	cp $< $@

plugin/rootfs/docker-volume-glusterfs:	bin/linux/docker-volume-glusterfs
	cp $< $@

build: plugin plugin/config.json plugin/rootfs/docker-volume-glusterfs
	@echo "[MAKE] Creating docker volume plugin..."
	@docker -c default plugin disable --force $(plugin) ; true
	@docker -c default plugin rm --force $(plugin) ; true

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

commit:
	git push

deploy:
	docker -c $(context) plugin install --alias $(alias) --grant-all-permissions $(plugin) GFS_SERVERS=$(servers) GFS_VOLUME=$(volume)