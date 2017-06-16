package main

import (
	"github.com/loginoff/nutanix-backup/nutanixapi"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
)

type NutanixMounter struct {
	ntnx       *nutanixapi.Client
	nfs_server string
	mount_root string
	Containers map[string]*NutanixContainer
}

type NutanixContainer struct {
	Name    string
	Mounted bool
}

func NewNutanixMounter(ntnx *nutanixapi.Client, nfsurl, local_mount_path string) *NutanixMounter {
	info, err := os.Stat(local_mount_path)
	if err != nil {
		log.Fatal(err)
	}
	if !info.IsDir() {
		log.Fatalf("%s must be a directory", local_mount_path)
	}

	return &NutanixMounter{
		ntnx:       ntnx,
		nfs_server: nfsurl,
		mount_root: local_mount_path,
		Containers: make(map[string]*NutanixContainer),
	}
}

func (m *NutanixMounter) GetContainerMountPathByUUID(UUID string) (string, error) {
	if cont, ok := m.Containers[UUID]; ok {
		log.Debugf("container mount for %s cached", cont.Name)
		return filepath.Join(m.mount_root, cont.Name), nil

	} else {
		cname, err := m.ntnx.GetContainerNameByUUID(UUID)
		if err != nil {
			log.Warning("Unable to retreive name for container %s", UUID)
			return "", err
		}

		return m.Mount(UUID, cname)
	}
}

func (m *NutanixMounter) Mount(UUID, cname string) (string, error) {
	mountpath := filepath.Join(m.mount_root, cname)
	log.Infof("Mounting %s:/%s to %s", m.nfs_server, cname, mountpath)

	if !IsDir(mountpath) {
		log.Debugf("%s does not exist, creating...", mountpath)
		if err := Mkdir(mountpath); err != nil {
			log.Fatal(err)
		}
	}

	if IsMounted(mountpath) {
		log.Fatalf("%s is already mounted", mountpath)
	}

	err := runCMD("mount", "-t", "nfs", "-o", "ro", m.nfs_server+":/"+cname, mountpath)
	if err != nil {
		log.Fatal(err)
	}
	m.Containers[UUID] = &NutanixContainer{cname, true}

	return mountpath, err
}

func (m *NutanixMounter) UmountAll() error {
	for _, cont := range m.Containers {
		mountpath := filepath.Join(m.mount_root, cont.Name)
		err := runCMD("umount", mountpath)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Unmounted %s", mountpath)
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return false
	}
	return true
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func IsMounted(path string) bool {
	err := runCMD("mountpoint", "-q", path)
	if err != nil {
		return false
	}
	return true
}

func Mkdir(path string) error {
	return os.MkdirAll(path, 0750)
}
