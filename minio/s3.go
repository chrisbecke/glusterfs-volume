package s3

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/go-plugins-helpers/volume"
	"volume-plugin/gfapi"
)

//------------------------------

// default file mode to create new volume directories in gluster
const defaultMode = 0755
const mountMode = 0555

// -------------------------------------------
// glusterFSDriver implementation

type mount struct {
	connections int
	name        string
	mountpoint  string
	createdAt   time.Time
	ids         map[string]int
}

type minioParams struct {
	showHidden bool
}

type driver struct {
	sync.RWMutex

	root string

	mounts map[string]*mount
}

// driver //////////////////

// mountpoint of a docker volume
func (d *driver) mountpoint(Name string) string {
	return filepath.Join(d.root, Name)
}

// subdir in the gluster volume for the docker volume
func (d *driver) subdir(Name string) string {
	return filepath.Join("/", Name)
}

// minio driver //////////////////

type minioDriver struct {
	driver

	config minioParams
}

// API volumeDriver.Create
func (d *minioDriver) Create(r *volume.CreateRequest) error {

	err := fmt.Errorf("Not creating %s - create operation not supported with options: %v", r.Name, r.Options)

	log.Printf("GlusterFS: %s", err.Error)

	return err
}

// volumeDriver.List
func (d *minioDriver) List() (*volume.ListResponse, error) {
	d.Lock()
	defer d.Unlock()

	var vols []*volume.Volume

	// TODO: enumerate buckets

	return &volume.ListResponse{Volumes: vols}, nil
}

// volumeDriver.Get
func (d *minioDriver) Get(r *volume.GetRequest) (*volume.GetResponse, error) {
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

	// TODO: lookup info about the bucket

	return &volume.GetResponse{}, nil
}

// volumeDriver.Remove
func (d *minioDriver) Remove(r *volume.RemoveRequest) error {

	err := fmt.Errorf("Remove operation not supported. r: %v", r)
	log.Printf("Minio: %s", err.Error)

	return err
}

// Volumedriver.Path
func (d *minioDriver) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	d.Lock()
	defer d.Unlock()

	v, ok := d.mounts[r.Name]
	if ok {
		return &volume.PathResponse{Mountpoint: v.mountpoint}, nil
	}

	err := fmt.Errorf("Volume not mounted. r: %v", r)
	log.Print(err.Error)

	return &volume.PathResponse{}, err
}

// VolumeDDriver.Mount
func (d *minioDriver) Mount(r *volume.MountRequest) (*volume.MountResponse, error) {
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

	shouldMount := v.connections == 0

	fi, err := os.Lstat(mountpoint)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(mountpoint, defaultMode); err != nil {
			log.Printf("os.MkDirAll(%s) Error: %v", mountpoint, err)
			return &volume.MountResponse{}, err
		}
	} else if err != nil {
		d.unmount(mountpoint)
		shouldMount = true
	}

	if fi != nil && !fi.IsDir() {
		err = fmt.Errorf("%v already exist and it's not a directory", mountpoint)
		log.Printf("os.Lstat(%s) Error: %v", mountpoint, err)
		return &volume.MountResponse{}, err
	}

	if shouldMount {

		d.mount(mountpoint, r.Name)

		if err != nil {
			return &volume.MountResponse{}, err
		}
		v.mountpoint = mountpoint
	}
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

		cmd := exec.Command("umount", v.mountpoint)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Exec failed: %v (%s)", err, output)
		}
		delete(d.mounts, r.Name)
	}

	return nil
}

func (d *minioDriver) Capabilities() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "global"}}
}

// -------------
// Utility functions

// connect to a mount. remember to defer vol.Unmount()
func (d *minioDriver) connect() (*gfapi.Volume, error) {

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

func (d *minioDriver) mount(mountpoint string, name string) error {

	cmd := exec.Command("glusterfs")

	for _, server := range d.config.hosts {
		cmd.Args = append(cmd.Args, "--volfile-server", server)
	}

	cmd.Args = append(cmd.Args, "--volfile-id", d.config.volume)

	cmd.Args = append(cmd.Args, "--subdir-mount", d.subdir(name))

	cmd.Args = append(cmd.Args, mountpoint)

	log.Printf("Executing %#v", cmd)

	_, err := cmd.CombinedOutput()

	return err
}

func (d *glusterfsDriver) unmount(mountpoint string) error {
	cmd := exec.Command("umount", mountpoint)
	_, err := cmd.CombinedOutput()
	return err
}

// volume from a glusterfs directory entry
func (d *glusterfsDriver) volume(stat os.FileInfo) (*volume.Volume, error) {
	if !stat.IsDir() {
		return &volume.Volume{}, fmt.Errorf("Object %s is not a directory", stat.Name())
	}

	return &volume.Volume{
		Name:      stat.Name(),
		CreatedAt: stat.ModTime().Format(time.RFC3339),
		//		Mountpoint: d.mountpoint(stat.Name()),
	}, nil
}

func (d *glusterfsDriver) findVolume(name string) (*volume.Volume, bool) {

	v, ok := d.mounts[name]
	if !ok {
		return nil, false
	}

	return &volume.Volume{
		Name: name,
		//		CreatedAt:   fmt.Sprintf(createdAt.Format(time.RFC3339)),,
		Mountpoint: v.mountpoint,
		//		Status: {}
	}, true
}
