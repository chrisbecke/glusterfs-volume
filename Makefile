registry = 
plugin = $(registry)$(notdir $(CURDIR))
context = default
node = node
go_source := main.go driver.go gfs.go
volume := test
# gv0
servers := lab717.mgsops.net,lab718.mgsops.net,lab719.mgsops.net
version = 4

docker-compose = DOCKER_BUILDKIT=1
docker-compose += COMPOSE_DOCKER_CLI_BUILD=1
docker-compose += docker-compose --context $(context)

docker = docker
docker += --context $(context)

.PHONY: build clean image test go

default: shell

all: build

bin/linux/docker-volume-glusterfs: $(go_source)
	@echo "[MAKE] Compiling glusterfs binary..."
	@$(docker-compose) --context default run builder go build -o ./bin/linux/docker-volume-glusterfs

image: bin/linux/docker-volume-glusterfs
	@echo "[MAKE] Building docker image for plugin..."
	@DOCKER_BUILDKIT=1 \
	docker build -t $(plugin) .

gluster_id.txt:
	@echo "[MAKE] Creating new instance of $(plugin)"
	$(MAKE) image
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
	@docker --context default plugin disable --force $(plugin) ; true
	@docker --context default plugin rm --force $(plugin) ; true

	sudo docker plugin create $(plugin) ./plugin/

clean:
	@echo "[CLEAN] Removing container $(id)"
	@docker --context default rm -vf "$(id)" | true
	rm -f ./gluster_id.txt | true
	@echo "[CLEAN] Disabling Plugin $(plugin)"
	docker --context default plugin disable -f $(plugin) | true
	@echo "[CLEAN] Stopping builder"
	docker-compose --context default down -v
	@echo "[CLEAN] Removing Plugin files"
	sudo rm -rf ./plugin
	rm -rf ./bin/linux/*

push:
	@echo "[PUSH] Pushing $(plugin)..."
	@docker --context default plugin push $(plugin)

test: test-gluster-vol1
	@-docker plugin disable $(plugin) --force
	@docker plugin set $(plugin) GFS_VOLUME=$(volume) GFS_SERVERS=$(servers)
	@docker plugin enable $(plugin)

commit:
	git push

deploy:
	#@docker -c $(context) node update --availability drain $(node)
	#@echo Proceeding with...
	@docker -c $(node) ps
	@-docker -c $(node) plugin disable $(alias) --force
	@-docker -c $(node) plugin rm $(alias) --force
	@-docker -c $(node) plugin install --alias $(alias) --grant-all-permissions $(plugin) GFS_SERVERS=$(servers) GFS_VOLUME=$(volume)
	#@docker -c $(context) node update --availability active $(node)

shell:
	docker-compose -c $(context) run builder

test-gluster-vol1: test-gluster-up
	@-$(docker-compose) exec glusterfs gluster volume create $(volume) glusterfs:/data/$(volume)
	@-$(docker-compose) exec glusterfs gluster volume start $(volume)

test-gluster-up:
	@$(docker-compose) up -d glusterfs

dockerlog:
	tail -f ~/Library/Containers/com.docker.docker/Data/log/vm/dockerd.log

count:
	docker ps -q | wc -l
