package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/loginoff/nutanix-backup/nutanixapi"

	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/configor"
)

var (
	username   *string
	password   *string
	configfile *string
	debug      *bool
	help       *bool
	mounter    *NutanixMounter
)

var BackupConfig struct {
	Prism_host         string
	Backup_root        string
	Nutanix_mount_root string
	Nutanix_cvm_addr   string

	VMs []VMBackup
}

type VMBackup struct {
	Name           string
	Disks          []string
	SizeEstimation int64
	VMInfo         nutanixapi.AHVVM
}

func (VM *VMBackup) EstimateBackupSize() string {
	var total int64
	for _, disk := range VM.Disks {
		for _, diskinfo := range VM.VMInfo.Config.VMDisks {
			if disk == fmt.Sprintf("%s.%d", diskinfo.Addr.DeviceBus, diskinfo.Addr.DeviceIndex) {
				total += diskinfo.VMDiskSize
			}
		}
	}

	VM.SizeEstimation = total
	return formatBytes(total)
}

func formatBytes(bytes int64) string {
	gigabyte := int64(1024 * 1024 * 1024)
	megabyte := int64(1024 * 1024)
	if bytes < gigabyte {
		return fmt.Sprintf("%dMB", bytes/megabyte)

	} else {
		return fmt.Sprintf("%dGB", bytes/gigabyte)

	}
}

const defaultconfig = "/root/bin/backupconf.yml"

func init() {
	username = flag.String("username", "", "Nutanix PRISM username")
	password = flag.String("password", "", "Nutanix PRISM password")
	configfile = flag.String("config", defaultconfig, "Configuration file for the entire backup process")
	debug = flag.Bool("debug", false, "Turn on debug logging")
	help = flag.Bool("help", false, "Display help")
}

func evaluateConfig() {
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(1)
	}

	//username
	if *username == "" {
		log.Warn("option '--username=' is not set  Default: admin is used")
		*username = "admin"
	}

	//password
	if *password == "" {
		log.Warn("option '--password=' is not set  Default: nutanix/4u is used")
		*password = "nutanix/4u"
	}

	if *configfile == defaultconfig {
		log.Infof("Using default configuration file %s", *configfile)
	}
	err := configor.Load(&BackupConfig, *configfile)
	if err != nil {
		log.Fatal(err)
	}

	if BackupConfig.Prism_host == "" {
		log.Fatalf("Must specify prism_host in %s", *configfile)
	}

	if len(BackupConfig.VMs) < 1 {
		log.Fatalf("Specify at least 1 VM to be backed up in %s", *configfile)
	}

	if BackupConfig.Nutanix_cvm_addr == "" {
		log.Fatalf("Must specify a CVM IP using nutanix_cvm_addr in %s", *configfile)
	}

	if BackupConfig.Backup_root == "" {
		log.Fatalf("Must specify a local directory for VM images using backup_root in %s", *configfile)
	}

	info, err := os.Stat(BackupConfig.Backup_root)
	if err != nil {
		log.Fatal(err)
	}
	if !info.IsDir() {
		log.Fatalf("%s must be a directory", BackupConfig.Backup_root)
	}
}

func setupLogging() {
	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	logfile := "nutanix_backup.log"
	f, err := os.OpenFile(logfile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)
	if err != nil {
		log.Fatalf("Error opening file %s", logfile)
	}
	log.SetOutput(io.MultiWriter(f, os.Stderr))
}

func BackupVM(ntnx *nutanixapi.Client, vm *VMBackup) error {
	log.Infof("Starting with the backup of %s", vm.Name)

	if len(vm.Disks) < 1 {
		log.Infof("No disks specified to backup for VM %s", vm.Name)
		log.Infof("Skipping VM %s ...", vm.Name)
		return nil
	}
	ahvvm := vm.VMInfo

	log.Infof("Creating a snapshot of %s (%s)", ahvvm.Config.Name, ahvvm.UUID)

	snapshot_name := getSnapshotName(vm.Name)

	taskUUID, err := ntnx.CreateVMSnapshot(ahvvm.UUID, snapshot_name)
	snapshot_task, err := ntnx.PollTaskForCompletion(taskUUID)
	if err != nil {
		return err
	}

	//Get snapshot info from the task
	var snapshot_uuid string
	for _, entity := range snapshot_task.EntityList {
		if entity.EntityType == "Snapshot" && entity.EntityName == snapshot_name {
			snapshot_uuid = entity.UUID
			break
		}
	}
	log.Debugf("Created snapshot %s", snapshot_uuid)

	snapshot_info, err := ntnx.GetSnapshotByUUID(snapshot_uuid)
	if err != nil {
		return err
	}

	//A little sanity check
	if snapshot_info.SnapshotName != snapshot_name ||
		snapshot_info.VMUUID != ahvvm.UUID ||
		snapshot_info.VMCreateSpecification.Name != vm.Name {
		log.Warningf("Snapshot name %s", snapshot_info.SnapshotName)
		log.Warningf("Snapshot VM UUID %s", snapshot_info.UUID)
		log.Warningf("Snapshot VM Name %s", snapshot_info.VMCreateSpecification.Name)
		log.Warningf("AHV VM UUID %s", ahvvm.UUID)
		return fmt.Errorf("Discrepancies in snapshot info")
	}

	backup_path := filepath.Join(BackupConfig.Backup_root, snapshot_name)
	if err := os.MkdirAll(backup_path, 0750); err != nil {
		log.Debug("Unable to create directory for backups")
		return err
	}

	if err := WriteSnapshotInfo(snapshot_info, filepath.Join(backup_path, "ahv_vm")); err != nil {
		log.Errorf("Unable to write snapshot info for %s", vm.Name)
		return err
	}

	//For each vdisk to be backed up, find it in the snapshot
	for _, disk := range vm.Disks {
		disk_uuid := ""
		container_uuid := ""
		for _, vdisk := range snapshot_info.VMCreateSpecification.VMDisks {
			if disk == fmt.Sprintf("%s.%d", vdisk.DiskAddress.DeviceBus, vdisk.DiskAddress.DeviceIndex) {
				disk_uuid = vdisk.VMDiskClone.VMDiskUUID
				container_uuid = vdisk.VMDiskClone.ContainerUUID
				break
			}
		}

		if disk_uuid == "" || container_uuid == "" {
			log.Errorf("Unable to find VM %s disk %s in snapshot %s", vm.Name, disk, snapshot_info.UUID)
			return fmt.Errorf("Unable to find all disks to backup for VM %s", vm.Name)
		}

		disk_container_path := fmt.Sprintf(".acropolis/snapshot/%s/vmdisk/%s", snapshot_info.GroupUUID, disk_uuid)
		log.Debugf("Starting backup of %s", disk_container_path)
		err := BackupVDisk(container_uuid, disk_container_path, backup_path, disk)
		if err != nil {
			return err
		}
	}

	//After all disks are successfully backed up, delete the snapshot
	if snapshot_info.SnapshotName != snapshot_name {
		log.Fatal("Wrong snapshot")
	}

	delete_task, err := ntnx.DeleteVMSnapshotByUUID(snapshot_info.UUID)
	if err != nil {
		log.Warning("Error initiating snapshot deletion %s", snapshot_info.UUID)
		return err
	}

	snapshot_task, err = ntnx.PollTaskForCompletion(delete_task)
	if err != nil {
		log.Warning("Trouble waiting for task %s", snapshot_task.UUID)
		return err
	}

	return err
}

func WriteSnapshotInfo(spec *nutanixapi.AHVSnapshotInfo, path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	e.SetIndent("", "\t")
	return e.Encode(spec)
}

func BackupVDisk(container_UUID, disk_container_path, vm_root, disk_name string) error {
	container_root, err := mounter.GetContainerMountPathByUUID(container_UUID)
	if err != nil {
		return err
	}
	vdisk_path := filepath.Join(container_root, disk_container_path)
	backup_path := filepath.Join(vm_root, disk_name)
	log.Infof("Backing up %s to %s", vdisk_path, backup_path)
	return runCMD("rsync", "-P", "--sparse", vdisk_path, backup_path)
}

func getSnapshotName(vmname string) string {
	t := time.Now()

	return fmt.Sprintf("%s_backup_%s", vmname, t.Format("20060102_1504"))
}

func runCMD(cmd string, args ...string) (err error) {
	proc := exec.Command(cmd, args...)
	proc.Stdin = os.Stdin
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr

	return proc.Run()
}

func main() {
	evaluateConfig()
	setupLogging()

	ntnx, err := nutanixapi.NewClient(BackupConfig.Prism_host, *username, *password, false)
	if err != nil {
		log.Fatal(err)
	}

	//Populate VM list with VM information
	allvms, err := ntnx.GetVMs()
	if err != nil {
		log.Fatalf("Unable to retrieve VM list from PRISM %s", err)
	}
	for vmidx, vm := range BackupConfig.VMs {
		for _, nvm := range allvms {
			if vm.Name == nvm.Config.Name {
				if vm.VMInfo.UUID != "" {
					log.Fatalf("More than one VM found with the name %s, aborting", vm.Name)
				} else {
					BackupConfig.VMs[vmidx].VMInfo = nvm
				}
			}
		}
		if BackupConfig.VMs[vmidx].VMInfo.UUID == "" {
			log.Fatalf("Did not find a VM named %s, aborting", vm.Name)
		}
	}

	var totalSize int64
	for _, vm := range BackupConfig.VMs {
		fmt.Printf("%20s (%d disks, %s total)\n", vm.Name, len(vm.Disks), vm.EstimateBackupSize())
		totalSize += vm.SizeEstimation
	}

	if !askForConfirmation(fmt.Sprintf("Backup these %d VMs, totalling %s?\n", len(BackupConfig.VMs), formatBytes(totalSize))) {
		log.Info("User cancelled backup")
		os.Exit(1)
	}

	mounter = NewNutanixMounter(ntnx, BackupConfig.Nutanix_cvm_addr, BackupConfig.Nutanix_mount_root)
	defer mounter.UmountAll()

	for _, vm := range BackupConfig.VMs {
		err = BackupVM(ntnx, &vm)
		if err != nil {
			log.Fatalf("Failed to backup VM %s", vm.Name)
		}
	}
}

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
