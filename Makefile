REPOSITORY_BASE = chrisbecke
MODULE = glusterfs-plugin
VERSION = $(shell git describe --tags)
PLUGIN = $(REPOSITORY_BASE)/$(MODULE):$(VERSION)

FILES := src/main.go
FILES += src/glusterfs/driver.go
FILES += src/glusterfs/gfs.go

volume := test
servers := lab717.mgsops.net,lab718.mgsops.net,lab719.mgsops.net

DOCKER = docker
DOCKER_COMPOSE = $(DOCKER) compose

.PHONY: build clean image test go

default: shell

all: build

bin/rootfs.tar: Dockerfile
	@echo [MAKE] Making rootfs
	@mkdir -p bin
	@$(DOCKER) buildx build --output type=tar,dest=$@ .

src/$(MODULE): $(FILES)
	@echo "[MAKE] Compiling glusterfs binary..."
	@$(DOCKER_COMPOSE) run builder go build


plugin/rootfs/usr/local/bin/$(MODULE): src/$(MODULE)
	@echo "[Make] Copying executable into plugin"
	@sudo cp $< $@

plugin/rootfs: bin/rootfs.tar
	@echo [MAKE] Expanding rootfs.tar
	sudo rm -rf $@
	mkdir -p $@
	sudo tar -xpf bin/rootfs.tar -C $@

plugin:	plugin/rootfs plugin/config.json

plugin/config.json:	config.json
	@jq '(.description = .description + " " + "$(VERSION)")' $< > $@

build: plugin plugin/config.json

clean:
	@echo "[CLEAN] Removing Plugin files"
	@sudo rm -rf ./plugin
	@rm -rf ./bin/*

push:
	@echo "[PUSH] Pushing $(plugin)..."
	@$(DOCKER) plugin push $(plugin)

commit:
	git push

deploy:
#	@docker -c $(context) node update --availability drain $(node)
#	@echo Proceeding with...
	@docker -c $(node) ps
	@-docker -c $(node) plugin disable $(alias) --force
	@-docker -c $(node) plugin rm $(alias) --force
	@-docker -c $(node) plugin install --alias $(alias) --grant-all-permissions $(plugin) GFS_SERVERS=$(servers) GFS_VOLUME=$(volume)
	#@docker -c $(context) node update --availability active $(node)

shell:
	$(DOCKER_COMPOSE) run builder

test-gluster-vol1: test-gluster-up
	@-$(docker-compose) exec glusterfs gluster volume create $(volume) glusterfs:/data/$(volume)
	@-$(docker-compose) exec glusterfs gluster volume start $(volume)

test-gluster-up:
	@$(docker-compose) up -d glusterfs

dockerlog:
	tail -f ~/Library/Containers/com.docker.docker/Data/log/vm/dockerd.log

count:
	$(DOCKER) ps -q | wc -l

plugin-start:
	docker plugin set $(PLUGIN) GFS_SERVERS=$(servers) GFS_VOLUME=$(volume)
	docker plugin enable $(PLUGIN)

plugin-create: plugin/rootfs plugin/config.json plugin/rootfs/usr/local/bin/$(MODULE)
	@echo "[MAKE] Creating docker volume plugin..."
	@$(DOCKER) plugin disable --force $(PLUGIN) ; true
	@$(DOCKER) plugin rm --force $(PLUGIN) ; true
	sudo $(DOCKER) plugin create $(PLUGIN) ./plugin/

# Installs the plugin for local dev
install: plugin-create

build: plugin/rootfs plugin/config.json plugin/rootfs/usr/local/bin/$(MODULE)

test: plugin-start
