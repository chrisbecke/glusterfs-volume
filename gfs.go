package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gluster/gogfapi/gfapi"
)

// -------------------------------------------
// glusterFSDriver implementation

// holds config for glfs
type glfsConnector struct {
	conn *gfapi.Volume

	volume string

	hosts []string
}

//////////////////////////////////////////////////////////////////////////////////
//

func (d *glfsConnector) removeContent(path string) error {

	dir, err := d.conn.Open(path)
	if err != nil {
		log.Printf("RemoveAll error. gogfapi.Open(%s), err: %v", path, err)
		return err
	}
	defer dir.Close()

	files, err := dir.Readdir(0)
	if err != nil {
		log.Printf("RemoveAll error. gogfapi.Readdir(0) path: '%s', err: %v", path, err)
		return err
	}

	for _, file := range files {
		name := file.Name()
		subdir := filepath.Join(path, name)
		if name == "." || name == ".." {
			// do nothing
		} else if file.IsDir() {
			if err := d.removeAll(subdir); err != nil {
				return err
			}
		} else {
			if err := d.conn.Unlink(subdir); err != nil {
				log.Printf("RemoveAll error. gogfapi.Unlink('%s'), err: %v", subdir, err)
			}
		}
	}
	return nil
}

func (d *glfsConnector) removeAll(path string) error {
	if err := d.removeContent(path); err != nil {
		return err
	}
	if err := d.conn.Rmdir(path); err != nil {
		log.Printf("RemoveAll error. gogfapi.Rmdir(%s), err: %v", path, err)
		return err
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// -------------
// Utility functions

func (d *glfsConnector) create(name string) error {

	subdir := filepath.Join("/", name)

	err := d.conn.Mkdir(subdir, defaultMode)

	if err != nil {
		log.Printf("gogfapi error. Mkdir dir: '%s'. err: %v", subdir, err)
	}

	return err
}

func (d *glfsConnector) list() ([]os.FileInfo, error) {

	dir, err := d.conn.Open(".")
	if err != nil {
		log.Printf("gogfapi error. Open dir: '.'. err: %v", err)
		return nil, err
	}
	defer dir.Close()

	dirs, err := dir.Readdir(0)
	if err != nil {
		log.Printf("gogfapi error. Readdir(0) dir: '.'. err: %v", err)
		return nil, err
	}

	return dirs, nil
}

func (d *glfsConnector) get(name string) (os.FileInfo, error) {

	subdir := filepath.Join("/", name)

	stat, err := d.conn.Stat(subdir)
	if err != nil {
		// This is an expected error, as docker calls getPath optimistically when creating
		// volumes to test if they exist
		//		log.Printf("gogfapi error. Stat('%s'). err: %v", subdir, err)
		return nil, err
	}

	if !stat.IsDir() {
		err = fmt.Errorf("Should be a directory: %s", name)
		log.Printf("glusterfs config error. Expected a directory: %s, got: %v", name, stat)
		return nil, err
	}

	return stat, nil
}

func (d *glfsConnector) remove(name string) error {

	subdir := filepath.Join("/", name)

	err := d.removeAll(subdir)

	return err
}

// ensureMount
func (d *glfsConnector) mountWithGlusterfs(mountpoint string, name string) error {

	cmd := exec.Command("glusterfs")

	for _, server := range d.hosts {
		cmd.Args = append(cmd.Args, "--volfile-server", server)
	}

	cmd.Args = append(cmd.Args, "--volfile-id", d.volume)
	cmd.Args = append(cmd.Args, "--subdir-mount", filepath.Join("/", name))
	cmd.Args = append(cmd.Args, mountpoint)

	log.Printf("Executing %v", cmd)

	_, err := cmd.CombinedOutput()

	return err
}

// ensureMount
func (d *glfsConnector) mountWithMount(mountpoint string, name string) error {

	cmd := exec.Command("mount")
	cmd.Args = append(cmd.Args, "-t", "glusterfs")
	server := d.hosts[rand.Intn(len(d.hosts))]
	url := fmt.Sprintf("%s:/%s/%s", server, d.volume, name)
	cmd.Args = append(cmd.Args, url)
	cmd.Args = append(cmd.Args, mountpoint)
	log.Printf("Executing %#v", cmd)
	_, err := cmd.CombinedOutput()

	return err
}

func (d *glfsConnector) unmount(mountpoint string) error {

	cmd := exec.Command("umount", mountpoint)
	_, err := cmd.CombinedOutput()

	return err
}
