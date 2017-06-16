package nutanixapi

type AHVVM struct {
	UUID             string `json:"uuid"`
	LogicalTimestamp int    `json:"logicalTimestamp"`
	Config           struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		NumVcpus        int    `json:"numVcpus"`
		NumCoresPerVcpu int    `json:"numCoresPerVcpu"`
		MemoryMb        int    `json:"memoryMb"`
		VMDisks         []struct {
			Addr struct {
				DeviceBus   string `json:"deviceBus"`
				DeviceIndex int    `json:"deviceIndex"`
			} `json:"addr"`
			IsCdrom           bool   `json:"isCdrom"`
			IsEmpty           bool   `json:"isEmpty"`
			SourceImage       string `json:"sourceImage"`
			IsSCSIPassthrough bool   `json:"isSCSIPassthrough"`
			ID                string `json:"id"`
			VMDiskUUID        string `json:"vmDiskUuid,omitempty"`
			SourceVMDiskUUID  string `json:"sourceVmDiskUuid,omitempty"`
			ContainerID       int    `json:"containerId,omitempty"`
			ContainerUUID     string `json:"containerUuid,omitempty"`
			VMDiskSize        int64  `json:"vmDiskSize,omitempty"`
		} `json:"vmDisks"`
		VMNics []struct {
			MacAddress  string `json:"macAddress"`
			NetworkUUID string `json:"networkUuid"`
			Model       string `json:"model"`
		} `json:"vmNics"`
	} `json:"config"`
	HostUUID string `json:"hostUuid,omitempty"`
	State    string `json:"state"`
}

type APIResponse_VMS struct {
	Metadata struct {
		GrandTotalEntities int `json:"grandTotalEntities"`
		TotalEntities      int `json:"totalEntities"`
	} `json:"metadata"`
	Entities []AHVVM `json:"entities"`
}

type AHVSnapshotSpec struct {
	VMUuid       string `json:"vmUuid"`
	SnapshotName string `json:"snapshotName"`
}

type AHVSnapshotSpecList struct {
	SnapshotSpecs []AHVSnapshotSpec `json:"snapshotSpecs"`
}

type AHVSnapshotInfo struct {
	UUID                  string `json:"uuid"`
	Deleted               bool   `json:"deleted"`
	LogicalTimestamp      int    `json:"logicalTimestamp"`
	CreatedTime           int64  `json:"createdTime"`
	GroupUUID             string `json:"groupUuid"`
	VMUUID                string `json:"vmUuid"`
	SnapshotName          string `json:"snapshotName"`
	VMCreateSpecification struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		NumVcpus        int    `json:"numVcpus"`
		NumCoresPerVcpu int    `json:"numCoresPerVcpu"`
		MemoryMb        int    `json:"memoryMb"`
		VMDisks         []struct {
			DiskAddress struct {
				DeviceBus   string `json:"deviceBus"`
				DeviceIndex int    `json:"deviceIndex"`
			} `json:"diskAddress"`
			IsCdrom      interface{} `json:"isCdrom"`
			IsEmpty      interface{} `json:"isEmpty"`
			VMDiskCreate interface{} `json:"vmDiskCreate"`
			VMDiskClone  struct {
				VMDiskUUID      string      `json:"vmDiskUuid"`
				ImagePath       interface{} `json:"imagePath"`
				MinimumSize     interface{} `json:"minimumSize"`
				MinimumSizeMb   interface{} `json:"minimumSizeMb"`
				SnapshotGroupID interface{} `json:"snapshotGroupId"`
				ContainerUUID   string      `json:"containerUuid"`
				VmdiskUUID      string      `json:"vmdisk_uuid"`
				NdfsFilepath    interface{} `json:"ndfs_filepath"`
			} `json:"vmDiskClone"`
			IsScsiPassThrough interface{} `json:"isScsiPassThrough"`
			IsThinProvisioned interface{} `json:"isThinProvisioned"`
		} `json:"vmDisks"`
		VMNics []struct {
			MacAddress  string `json:"macAddress"`
			NetworkUUID string `json:"networkUuid"`
		} `json:"vmNics"`
	} `json:"vmCreateSpecification"`
}

type TaskInfo struct {
	UUID        string `json:"uuid"`
	MetaRequest struct {
		MethodName string `json:"methodName"`
	} `json:"metaRequest"`
	MetaResponse struct {
		Error       string `json:"error"`
		ErrorDetail string `json:"errorDetail"`
	} `json:"metaResponse"`
	CreateTime      int64 `json:"createTime"`
	StartTime       int64 `json:"startTime"`
	CompleteTime    int64 `json:"completeTime"`
	LastUpdatedTime int64 `json:"lastUpdatedTime"`
	EntityList      []struct {
		UUID       string `json:"uuid"`
		EntityType string `json:"entityType"`
		EntityName string `json:"entityName"`
	} `json:"entityList"`
	OperationType      string `json:"operationType"`
	Message            string `json:"message"`
	PercentageComplete int    `json:"percentageComplete"`
	ProgressStatus     string `json:"progressStatus"`
}
