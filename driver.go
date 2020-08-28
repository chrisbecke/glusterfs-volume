package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/gluster/gogfapi/gfapi"
)

//------------------------------

// default file mode to create new volume directories in gluster
const defaultMode = 0755
const mountMode = 0555
const showHidden = false

//////////////////////////////////////////////////////////////////////////////////
//

func RemoveContent(d *gfapi.Volume, path string) error {

	dir, err := d.Open(path)
	if err != nil {
		log.Printf("RemoveAll error. gogfapu.Open(%s), err: %v", path, err)
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
func (d *glfsParams) mount(mountpoint string, name string) error {

	subdir := filepath.Join("/", name)

	cmd := exec.Command("glusterfs")

	for _, server := range d.hosts {
		cmd.Args = append(cmd.Args, "--volfile-server", server)
	}

	cmd.Args = append(cmd.Args, "--volfile-id", d.volume)

	cmd.Args = append(cmd.Args, "--subdir-mount", subdir)

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

	config glfsParams
}

// API volumeDriver.Create
func (d *glusterfsDriver) Create(r *volume.CreateRequest) error {
	d.Lock()
	defer d.Unlock()

	err := d.config.create(r.Name)

	return err
}

// volumeDriver.List
func (d *glusterfsDriver) List() (*volume.ListResponse, error) {
	d.Lock()
	defer d.Unlock()

	files, err := d.config.list()

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

	// Find it if its listed in mounts.
	v, ok := d.mounts[r.Name]
	if ok {
		vol := &volume.Volume{
			Name:       r.Name,
			CreatedAt:  v.createdAt.Format(time.RFC3339),
			Mountpoint: v.mountpoint,
			//			Status: {}
		}

		return &volume.GetResponse{Volume: vol}, nil
	}

	stat, err := d.config.get(r.Name)
	if err != nil {
		return &volume.GetResponse{}, err
	}

	vo := &volume.Volume{
		Name:      stat.Name(),
		CreatedAt: stat.ModTime().Format(time.RFC3339),
		//		Mountpoint: d.mountpoint(stat.Name()),
		//		Status: {},
	}
	return &volume.GetResponse{Volume: vo}, nil
}

// volumeDriver.Remove
func (d *glusterfsDriver) Remove(r *volume.RemoveRequest) error {
	d.Lock()
	defer d.Unlock()

	v, ok := d.mounts[r.Name]
	if ok && v.connections != 0 {
		log.Printf("Error: %n Existing local mounts", v.connections)
	}

	err := d.config.remove(r.Name)

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

// VolumeDDriver.Mount
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

	if err := d.ensureMount(v, mountpoint, r.Name); err != nil {
		return &volume.MountResponse{}, err
	}

	v.mountpoint = mountpoint
	v.ids[r.ID]++
	v.connections++

	log.Printf("Mounted registration: %+v", v)

	return &volume.MountResponse{Mountpoint: mountpoint}, nil
}

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

		d.config.unmount(v.mountpoint)

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

func (d *glusterfsDriver) ensureMount(mount *activeMount, mountpoint string, name string) error {

	stat, err := os.Lstat(mountpoint)

	if err == nil && mount.connections > 0 {
		return nil
	}

	if os.IsNotExist(err) {
		if err := os.MkdirAll(mountpoint, defaultMode); err != nil {
			log.Printf("ensureMount error. os.MkdirAll %s, err: %v", mountpoint, err)
			return err
		}
	} else if err != nil {
		log.Printf("ensureMount is unmounting dodgey fuse mount: %v", err)
		d.config.unmount(mountpoint)
	}

	if stat != nil && !stat.IsDir() {
		err = fmt.Errorf("mountpoint is not a directory")
		log.Printf("ensureMount error: lstat %s, err: %v", mountpoint, err)
		return err
	}

	if err = d.config.mount(mountpoint, name); err != nil {
		log.Printf("ensureMount error: %v", err)
		return err
	}

	return nil
}
