package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-plugins-helpers/volume"
)

//------------------------------

// default file mode to create new volume directories in gluster
const defaultMode = 0755
const mountMode = 0555
const showHidden = false

///////////////////////////////////////////////////////////////////////////////

// ActiveMount holds active mounts
type activeMount struct {
	connections int
	mountpoint  string
	createdAt   time.Time
	ids         map[string]int
}

type glusterfsDriver struct {
	sync.RWMutex

	root string

	mounts map[string]*activeMount

	client glfsConnector
}

// API volumeDriver.Create
func (d *glusterfsDriver) Create(r *volume.CreateRequest) error {
	d.Lock()
	defer d.Unlock()

	err := d.client.create(r.Name)

	return err
}

// volumeDriver.List
func (d *glusterfsDriver) List() (*volume.ListResponse, error) {
	d.Lock()
	defer d.Unlock()

	files, err := d.client.list()

	if err != nil {
		return &volume.ListResponse{}, err
	}

	var vols []*volume.Volume
	for _, file := range files {
		if file.IsDir() && (showHidden || !strings.HasPrefix(file.Name(), ".")) {

			vols = append(vols, &volume.Volume{Name: file.Name()})
		}
	}

	return &volume.ListResponse{Volumes: vols}, nil
}

// volumeDriver.Get
func (d *glusterfsDriver) Get(r *volume.GetRequest) (*volume.GetResponse, error) {
	d.Lock()
	defer d.Unlock()

	s := make(map[string]interface{})
	s["gluster-volume-version"] = "3"

	// Find it if its listed in mounts.
	v, ok := d.mounts[r.Name]
	if ok {
		s["source"] = "mounts"
		vol := &volume.Volume{
			Name:       r.Name,
			CreatedAt:  v.createdAt.Format(time.RFC3339),
			Mountpoint: v.mountpoint,
			Status:     s,
		}

		return &volume.GetResponse{Volume: vol}, nil
	}

	stat, err := d.client.get(r.Name)
	if err != nil {
		return &volume.GetResponse{}, err
	}
	s["source"] = "gogfs-statd"
	vo := &volume.Volume{
		Name:      stat.Name(),
		CreatedAt: stat.ModTime().Format(time.RFC3339),
		Status:    s,
	}
	return &volume.GetResponse{Volume: vo}, nil
}

// volumeDriver.Remove
func (d *glusterfsDriver) Remove(r *volume.RemoveRequest) error {
	d.Lock()
	defer d.Unlock()

	v, ok := d.mounts[r.Name]
	if ok && v.connections != 0 {
		log.Printf("Error: %d Existing local mounts", v.connections)
	}

	err := d.client.remove(r.Name)

	delete(d.mounts, r.Name)

	return err
}

// Volumedriver.Path
func (d *glusterfsDriver) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	d.Lock()
	defer d.Unlock()

	v, ok := d.mounts[r.Name]
	if !ok || v.connections == 0 || v.mountpoint == "" {
		err := fmt.Errorf("no mountpoint for volume.")
		log.Printf("Path error. name: %s, err: %v", r.Name, err)
		return &volume.PathResponse{}, err
	}

	return &volume.PathResponse{Mountpoint: v.mountpoint}, nil
}

// VolumeDriver.Mount
func (d *glusterfsDriver) Mount(r *volume.MountRequest) (*volume.MountResponse, error) {
	d.Lock()
	defer d.Unlock()

	log.Printf("GlusterFS: Mount %+v", r)

	mountpoint := d.mountpoint(r.Name)

	v, ok := d.mounts[r.Name]
	if !ok {
		v = &activeMount{
			mountpoint: mountpoint,
			ids:        map[string]int{},
		}
		d.mounts[r.Name] = v
	}

	stat, err := os.Lstat(mountpoint)

	if err != nil || v.connections == 0 {

		if os.IsNotExist(err) {
			if err := os.MkdirAll(mountpoint, defaultMode); err != nil {
				log.Printf("Mount error. os.MkdirAll %s, err: %v", mountpoint, err)
				return &volume.MountResponse{}, err
			}
		} else if err != nil {
			log.Printf("Mount is unmounting dodgey fuse mount: %v", err)
			d.client.unmount(mountpoint)
		}

		if stat != nil && !stat.IsDir() {
			err = fmt.Errorf("mountpoint is not a directory")
			log.Printf("Mount error: lstat %s, err: %v", mountpoint, err)
			return &volume.MountResponse{}, err
		}

		if err = d.client.mountWithGlusterfs(mountpoint, r.Name); err != nil {
			log.Printf("Mount error: %v", err)
			return &volume.MountResponse{}, err
		}
	}

	v.mountpoint = mountpoint
	v.ids[r.ID]++
	v.connections++

	log.Printf("Mounted registration: %+v", v)

	return &volume.MountResponse{Mountpoint: mountpoint}, nil
}

// VolumeDriver.Unmount
func (d *glusterfsDriver) Unmount(r *volume.UnmountRequest) error {
	log.Printf("GlusterFS: Unmount %v", r)

	v, ok := d.mounts[r.Name]
	if !ok {
		err := fmt.Errorf("Volume not found in active Mounts: %s", r.Name)
		log.Printf("Unmount failed: %v", err)
		return err
	}

	if v.connections == 0 {
		err := fmt.Errorf("Mount has no active connections: %s", r.Name)
		log.Printf("Unmount failed: %v", err)
		return err
	}

	i, ok := v.ids[r.ID]
	if !ok {
		err := fmt.Errorf("Mount %s does not know about this client ID: %s", r.Name, r.ID)
		log.Printf("Unmount failed: %v", err)
		return err
	}

	i--
	v.connections--

	if i <= 1 {
		delete(v.ids, r.ID)
	} else {
		v.ids[r.ID] = i
	}

	if len(v.ids) == 0 {
		log.Printf("Unmounting volume %s with %v clients", r.Name, v.connections)

		d.client.unmount(v.mountpoint)

		delete(d.mounts, r.Name)
	}

	return nil
}

func (d *glusterfsDriver) Capabilities() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "global"}}
}

// mountpoint of a docker volume
func (d *glusterfsDriver) mountpoint(Name string) string {
	return filepath.Join(d.root, Name)
}
