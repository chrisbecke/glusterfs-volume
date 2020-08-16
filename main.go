package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/gluster/gogfapi/gfapi"
)

//------------------------------

// config.json settings
const socketAddress = "/run/docker/plugins/glusterfs.sock"
const propogatedMount = "/mnt/volumes"

// default file mode to create new volume directories in gluster
const defaultMode = 0755

// -------------------------------------------
// glusterFSDriver implementation

// holds configfor glfs
type glfsParams struct {
	volume string
	hosts  []string
}

// ActiveMount holds active mounts
type activeMount struct {
	connections int
	mountpoint  string
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

	vol, err := d.connect()
	if err != nil {
		return err
	}
	defer vol.Unmount()

	subdir := d.subdir(r.Name)
	err = vol.Mkdir(filepath.Join(subdir), defaultMode)
	if err != nil {
		log.Printf("gogfapi.Mkdir('%s') Error %v", subdir, err)
	}

	return err
}

// volumeDriver.List
func (d *glusterfsDriver) List() (*volume.ListResponse, error) {
	d.Lock()
	defer d.Unlock()

	showHidden := false

	vol, err := d.connect()
	if err != nil {
		return &volume.ListResponse{}, err
	}
	defer vol.Unmount()

	dir, err := vol.Open(".")
	if err != nil {
		log.Printf("gogfapi.Open('.') Error: %v", err)
		return &volume.ListResponse{}, err
	}
	defer dir.Close()

	files, err := dir.Readdir(0)
	if err != nil {
		log.Printf("gogfapi.Readdir(0) Error: %v", err)
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

	vol, err := d.connect()
	if err != nil {
		return &volume.GetResponse{}, err
	}
	defer vol.Unmount()

	subdir := d.subdir(r.Name)

	stat, err := vol.Stat(subdir)
	if err != nil {
		log.Printf("gogfapi.Stat('%s') Error: %v", subdir, err)
		return &volume.GetResponse{}, err
	}

	v, err := d.volume(stat)
	if err != nil {
		return &volume.GetResponse{}, err
	}

	return &volume.GetResponse{Volume: v}, nil
}

// volumeDriver.Remove
func (d *glusterfsDriver) Remove(r *volume.RemoveRequest) error {
	d.Lock()
	defer d.Unlock()

	vol, err := d.connect()
	if err != nil {
		return err
	}
	defer vol.Unmount()

	// TODO: recursively delete everything here.
	subdir := d.subdir(r.Name)

	err = vol.Rmdir(subdir)
	if err != nil {
		log.Printf("gogfapi.Rmdir('%s') Error: %v", subdir, err)
	}

	return err
}

// Volumedriver.Path
func (d *glusterfsDriver) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	d.Lock()
	defer d.Unlock()

	return &volume.PathResponse{Mountpoint: d.mountpoint(r.Name)}, nil
}

// VolumeDDriver.Mount
func (d *glusterfsDriver) Mount(r *volume.MountRequest) (*volume.MountResponse, error) {
	d.Lock()
	defer d.Unlock()

	log.Printf("Entered Mount %+v", r)

	mountpoint := d.mountpoint(r.Name)

	v, ok := d.mounts[r.Name]
	if !ok {
		v = &activeMount{
			mountpoint: mountpoint,
			ids:        map[string]int{},
		}
		d.mounts[r.Name] = v
	}

	if v.connections == 0 {
		fi, err := os.Lstat(v.mountpoint)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(v.mountpoint, defaultMode); err != nil {
				log.Printf("os.MkDirAll(%s) Error: %v", v.mountpoint, err)
				return &volume.MountResponse{}, err
			}
		} else if err != nil {
			log.Printf("os.Lstat(%s) Error: %v", v.mountpoint, err)
			return &volume.MountResponse{}, err
		}

		if fi != nil && !fi.IsDir() {
			err = fmt.Errorf("%v already exist and it's not a directory", v.mountpoint)
			log.Printf("os.Lstat(%s) Error: %v", v.mountpoint, err)
			return &volume.MountResponse{}, err
		}

		cmd := exec.Command("glusterfs")

		for _, server := range d.config.hosts {
			cmd.Args = append(cmd.Args, "--volfile-server", server)
		}

		cmd.Args = append(cmd.Args, "--volfile-id", d.config.volume)

		cmd.Args = append(cmd.Args, "--subdir-mount", d.subdir(r.Name))

		cmd.Args = append(cmd.Args, mountpoint)

		log.Printf("Executing %#v", cmd)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return &volume.MountResponse{}, fmt.Errorf("Exec failed: %v (%s)", err, output)
		}
	}
	v.ids[r.ID]++
	v.connections++

	log.Printf("Mounted registration: %+v", v)

	return &volume.MountResponse{Mountpoint: mountpoint}, nil
}

func (d *glusterfsDriver) Unmount(r *volume.UnmountRequest) error {
	log.Printf("Entered Unmount %v", r)

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

		cmd := exec.Command("umount", v.mountpoint)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Exec failed: %v (%s)", err, output)
		}
		delete(d.mounts, r.Name)
	}

	return nil
}

func (d *glusterfsDriver) Capabilities() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "local"}}
}

// -------------
// Utility functions

// connect to a mount. remember to defer vol.Unmount()
func (d *glusterfsDriver) connect() (*gfapi.Volume, error) {

	vol := &gfapi.Volume{}
	if err := vol.Init(d.config.volume, d.config.hosts...); err != nil {
		log.Printf("gogfapi.Init Error: %v", err)
		return vol, err
	}

	if err := vol.Mount(); err != nil {
		log.Printf("gogfapi.Mount Error: %v", err)
		return vol, err
	}

	return vol, nil
}

// volume from a glusterfs directory entry
func (d *glusterfsDriver) volume(stat os.FileInfo) (*volume.Volume, error) {
	if !stat.IsDir() {
		return &volume.Volume{}, fmt.Errorf("Object %s is not a directory", stat.Name())
	}

	return &volume.Volume{
		Name:       stat.Name(),
		Mountpoint: d.mountpoint(stat.Name()),
	}, nil
}

// mountpoint of a docker volume
func (d *glusterfsDriver) mountpoint(Name string) string {
	return filepath.Join(d.root, Name)
}

// subdir in the gluster volume for the docker volume
func (d *glusterfsDriver) subdir(Name string) string {
	return filepath.Join("/", Name)
}

// -------------
// main

func main() {

	gfsvol := os.Getenv("GFS_VOLUME")
	gfsservers := strings.Split(os.Getenv("GFS_SERVERS"), ",")

	d := &glusterfsDriver{
		mounts: map[string]*activeMount{},
		root:   propogatedMount,
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
