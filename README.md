# Nutanix-backup

A small utility for backing up VMs and vdisks on a Nutanix Acropolis cluster.

The idea is to allow snapshotting VMs running on the Acropolis hypervisor and to copy the snapshot images over to another machine that can access the Nutanix cluster over NFS.

It works as follows:
* Connect to the Nutanix PRISM API endpoints and retrieve information about VMs to be backed up.
* Create snapshots of the VMs.
* Mount Nutanix storage containers, where the snapshots are stored, over NFS.
* Copy the snapshot images over NFS.
* Write details about the VM into a file `ahv_vm`.
* Delete snapshots.
* Unmount NFS mounts.

## Usage

* Install this tool on the machine were you want to store your backups.
* Your backup machine needs to have access to the Nutanix PRISM endpoint and a CVM.
* Make sure you whitelist your backup machine, so you can mount Nutanix storage containers over NFS. In PRISM go to the little cog icon (top right) -> Filesystem Whitelists -> add the address of your backup machine.
* Create a configuration file (see `backupconf.yml.example`) where you enumerate all the VMs and disks on these VMs, that you want to backup.

```nutanix-backup --username nutanix --password nutanix/4u -conf backupconf.yml```

## Manually restoring the images

* You'll need to get the images back to a Nutanix storage container somehow. You can either mount the storage over NFS (`mount -t nfs [CVM addr]:/container_name /mnt/nutanix`) or copy them over sftp (each CVM listens for sftp on port 2222). Create a directory on the Nutanix storage container as a "staging area" for the vdisk images.
* Once you have the images copied over, you have to create a new VM that mimics the configuration of the old one. You can refer to the old VM's configuration in the ahv_vm file.
* When you add disks to the new VM, choose "CLONE FROM ADSF FILE" and point to the path of the image files on the container that you copied over.
* Once you have the VM up and running, you can delete the images from the "staging area", as clones have been made from them.


## Building locally
* Install GO
* go get github.com/loginoff/nutanix-backup

## Building using Docker
* Just make sure you have a running Docker daemon
* Clone this repo
* Run `dockerized_build.sh` in the repo root

## TODO

* Implement a restore procedure
* Implement cleanup (deletion of snapshots, unmounting etc.) in case of interruption

## Disclaimer
Use at your own peril. In case you burn down your system and destroy all your data with this tool or any part of it's code, the author can in no way be held responsible.
No responsibility is taken for any possible use of this code.
