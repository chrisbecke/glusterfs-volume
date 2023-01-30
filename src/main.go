package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"glusterfs-plugin/glusterfs"

	"github.com/docker/go-plugins-helpers/volume"
)

//------------------------------

// config.json settings
const socketAddress = "/run/docker/plugins/glusterfs.sock"

// -------------
// main

func main() {

	log.SetFlags(0)

	gfsvol := os.Getenv("GFS_VOLUME")
	gfsservers := strings.Split(os.Getenv("GFS_SERVERS"), ",")

	fmt.Printf("starting glusterfs volume plugin on \\%s with %v\n", gfsvol, gfsservers)

	c, err := glusterfs.NewGlusterClient(gfsvol, gfsservers...)
	if err != nil {
		log.Fatal(err)
	}

	//	defer vol.Unmount()

	d := glusterfs.NewDriver(c)

	h := volume.NewHandler(d)

	//	fmt.Printf("calling ServeUnix(%s)\n",socketAddress)
	err = h.ServeUnix(socketAddress, 0)

	log.Print(err)

	return
}
