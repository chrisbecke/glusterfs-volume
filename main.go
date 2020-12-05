package main

import (
	"log"
	"os"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/gluster/gogfapi/gfapi"
)

//------------------------------

// config.json settings
const socketAddress = "/run/docker/plugins/glusterfs.sock"
const propagatedMount = "/mnt/volumes"

// -------------
// main

func init() {
	// squash timestamps in logging as this logging stream is always encapsulated in a larger one, with timestamps.
	log.SetFlags(0)
	logfile := os.Getenv("LOGFILE")

	if logfile != "" {
		f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}
}

func main() {

	gfsvol := os.Getenv("GFS_VOLUME")
	gfsservers := strings.Split(os.Getenv("GFS_SERVERS"), ",")

	vol := &gfapi.Volume{}
	if err := vol.Init(gfsvol, gfsservers...); err != nil {
		log.Printf("gogfapi Error. Init volume: '%s', servers: %v. err: %v", gfsvol, gfsservers, err)
		return
	}

	if err := vol.Mount(); err != nil {
		log.Printf("gogfapi Error. Mount volume: '%s', servers: %v. err: %v", gfsvol, gfsservers, err)
		return
	}
	defer vol.Unmount()

	d := &glusterfsDriver{
		mounts: map[string]*activeMount{},
		root:   propagatedMount,
		client: glfsConnector{
			conn:   vol,
			volume: gfsvol,
			hosts:  gfsservers,
		},
	}

	h := volume.NewHandler(d)

	log.Printf("GlusterFS Volume Plugin listening on %s", socketAddress)
	log.Printf("Using GlusterFS volume %s hosted on servers %v", gfsvol, gfsservers)
	err := h.ServeUnix(socketAddress, 0)

	log.Print(err)

	return
}
