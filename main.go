package main

import (
	"log"
	"os"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
)

//------------------------------

// config.json settings
const socketAddress = "/run/docker/plugins/glusterfs.sock"
const propagatedMount = "/mnt/volumes"

// -------------
// main

func main() {

	gfsvol := os.Getenv("GFS_VOLUME")
	gfsservers := strings.Split(os.Getenv("GFS_SERVERS"), ",")
	logfile := os.Getenv("LOGFILE")

	if logfile != "" {
		f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	d := &glusterfsDriver{
		mounts: map[string]*activeMount{},
		root:   propagatedMount,
		config: glfsParams{
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
