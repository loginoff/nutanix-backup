# Nutanix-backup

A small utility for backing up VMs and vdisks on a Nutanix Acropolis cluster.

The idea is to allow snapshotting VMs running on the Acropolis hypervisor and to copy the snapshot images over to another machine that can access the Nutanix cluster over NFS.

## Usage

* Install this tool on the machine were you want to store your backups.
* Your backup machine needs to have access to the Nutanix PRISM endpoint and a CVM.
* Make sure you whitelist your backup machine, so you can mount Nutanix storage containers over NFS. In PRISM go to the little cog icon (top right) -> Filesystem Whitelists -> add the address of your backup machine.
* Create a configuration file (see `backupconf.yml.example`) where you enumerate all the VMs and disks on these VMs, that you want to backup.

```nutanix-backup --username nutanix --password nutanix/4u -conf backupconf.yml```

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
No responsibility is taken for any other possible use of this code.
