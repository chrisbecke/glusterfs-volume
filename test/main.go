package main

import (
	"log"

	"volume-plugin/gfapi"
)

func main() {

	vol := &gfapi.Volume{}
	if err := vol.Init("gv0", "server1", "server2"); err != nil {
		// handle error
		log.Fatalf("Init Error: %v", err)
	}

	log.Print("vol initialized")

	if err := vol.Mount(); err != nil {
		// handle error
		log.Fatalf("Mount Error: %v", err)
	}
	defer vol.Unmount()

	f, err := vol.Create("testfile")
	if err != nil {
		// handle error
		log.Fatalf("Create Error: %v", err)
	}
	defer f.Close()

	if _, err := f.Write([]byte("hello")); err != nil {
		// handle error
		log.Fatalf("Write Error: %v", err)
	}

	d, err := vol.Open(".")
	if err != nil {
		log.Fatalf("Open Error: %v", err)
	}
	defer d.Close()

	i, err := d.Readdir(0)
	if err != nil {
		log.Fatalf("Readdir Error: %v", err)
	}

	for _, file := range i {
		if file.IsDir() {
			name := file.Name()
			log.Printf("found dir: %+v", name)
		} else {
			log.Printf("not a dir: %+v", file)
		}
	}

	_, err = vol.Stat("not exist")
	if err != nil {
		log.Printf("Stat err: %+v", err)
	}

	s, err := vol.Stat("afolder")
	if err != nil {
		log.Printf("Stat Error: %+v", err)
	}
	log.Printf("Stat('afolder' %+v", s)
	log.Printf("Mode: %s", s.Mode().String())
	log.Printf("Perm: %s", s.Mode().Perm().String())

	return
}

//Live Videogame Solution.
