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

//////////////////////////////////////////////////////////////////////////////////
//

func RemoveContent(d *gfapi.Volume, path string) error {

	dir, err := d.Open(path)
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
			if err := RemoveAll(d, subdir); err != nil {
				return err
			}
		} else {
			if err := d.Unlink(subdir); err != nil {
				log.Printf("RemoveAll error. gogfapi.Unlink('%s'), err: %v", subdir, err)
			}
		}
	}
	return nil
}

func RemoveAll(d *gfapi.Volume, path string) error {
	if err := RemoveContent(d, path); err != nil {
		return err
	}
	if err := d.Rmdir(path); err != nil {
		log.Printf("RemoveAll error. gogfapi.Rmdir(%s), err: %v", path, err)
		return err
	}
	return nil
}

// -------------------------------------------
// glusterFSDriver implementation

// holds config for glfs
type glfsParams struct {
	volume string
	hosts  []string
}

///////////////////////////////////////////////////////////////////////////////
// -------------
// Utility functions

// connect to a mount. remember to defer vol.Unmount()
func (d *glfsParams) connect() (*gfapi.Volume, error) {

	vol := &gfapi.Volume{}
	if err := vol.Init(d.volume, d.hosts...); err != nil {
		log.Printf("gogfapi Error. Init volume: '%s', servers: %v. err: %v", d.volume, d.hosts, err)
		return vol, err
	}

	if err := vol.Mount(); err != nil {
		log.Printf("gogfapi Error. Mount volume: '%s', servers: %v. err: %v", d.volume, d.hosts, err)
		return vol, err
	}

	return vol, nil
}

func (d *glfsParams) create(name string) error {

	vol, err := d.connect()
	if err != nil {
		return err
	}
	defer vol.Unmount()

	subdir := filepath.Join("/", name)

	err = vol.Mkdir(subdir, defaultMode)

	if err != nil {
		log.Printf("gogfapi error. Mkdir dir: '%s'. err: %v", subdir, err)
	}

	return err
}

func (d *glfsParams) list() ([]os.FileInfo, error) {

	vol, err := d.connect()
	if err != nil {
		return nil, err
	}
	defer vol.Unmount()

	dir, err := vol.Open(".")
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

func (d *glfsParams) get(name string) (os.FileInfo, error) {
	//	If its not found locally, look on the remote gluster volume
	vol, err := d.connect()
	if err != nil {
		return nil, err
	}
	defer vol.Unmount()

	subdir := filepath.Join("/", name)

	stat, err := vol.Stat(subdir)
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

func (d *glfsParams) remove(name string) error {
	vol, err := d.connect()
	if err != nil {
		return err
	}
	defer vol.Unmount()

	subdir := filepath.Join("/", name)

	err = RemoveAll(vol, subdir)

	return err
}

// subdir in the gluster volume for the docker volume
//func (d *glfsParams) subdir(Name string) string {
//	return filepath.Join("/", Name)
//}

// ensureMount
func (d *glfsParams) mountWithGlusterfs(mountpoint string, name string) error {

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
func (d *glfsParams) mountWithMount(mountpoint string, name string) error {

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

func (d *glfsParams) unmount(mountpoint string) error {

	cmd := exec.Command("umount", mountpoint)
	_, err := cmd.CombinedOutput()

	return err
}
