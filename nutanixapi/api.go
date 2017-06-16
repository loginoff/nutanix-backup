package nutanixapi

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

type Client struct {
	baseurl_v1    string
	baseurl_ahv   string
	base64authstr string
	httpclient    http.Client
}

func NewClient(host, username, password string, verify_ssl bool) (*Client, error) {
	transport := http.DefaultTransport
	if !verify_ssl {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	c := Client{
		base64authstr: base64.StdEncoding.EncodeToString([]byte(username + ":" + password)),
		httpclient: http.Client{
			Timeout:   time.Second * 10,
			Transport: transport,
		},
		baseurl_v1:  fmt.Sprintf("https://%s:9440/PrismGateway/services/rest/v1/", host),
		baseurl_ahv: fmt.Sprintf("https://%s:9440/api/nutanix/v0.8/", host),
	}
	req, _ := http.NewRequest("GET", c.baseurl_v1+"cluster", nil)
	err := c.do_request(req, nil)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Client) do_request(req *http.Request, parsefunc func(body []byte) error) error {
	req.Header.Set("Authorization", "Basic "+c.base64authstr)
	resp, err := c.httpclient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("Unauthorized request: %s", resp.Request.URL)
	}

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)

	if resp.StatusCode != 200 {
		log.WithFields(log.Fields{
			"ResponseBody": buf.String(),
		}).Errorf("Response with status %d received", resp.StatusCode)
		return fmt.Errorf("Response with status %d received", resp.StatusCode)
	}

	if parsefunc != nil {
		return parsefunc(buf.Bytes())
	} else {
		return nil
	}
}

func (c *Client) GetVMs() ([]AHVVM, error) {
	var vmlist []AHVVM
	req, _ := http.NewRequest("GET", c.baseurl_ahv+"vms?includeVMDiskSizes=true", nil)
	err := c.do_request(req, func(body []byte) error {
		var apiresponse APIResponse_VMS
		err := json.Unmarshal(body, &apiresponse)
		if err != nil {
			log.Debugf("Unable to decode API response for %s", req.URL)
			return err
		}
		vmlist = apiresponse.Entities
		return nil
	})
	return vmlist, err
}

func (c *Client) GetVMByName(name string) (*AHVVM, error) {
	vms, err := c.GetVMs()
	if err != nil {
		return nil, err
	}

	count := 0
	var last AHVVM
	for _, vm := range vms {
		if vm.Config.Name == name {
			last = vm
			count++
		}
	}
	if count == 0 {
		return nil, fmt.Errorf("No VM by name %s", name)
	}
	if count != 1 {
		return nil, fmt.Errorf("Found %d VMs with the name %s", count, name)
	}
	return &last, nil
}

func (c *Client) CreateVMSnapshot(vmUUID, snapshotName string) (TaskUUID string, err error) {
	vmspec := AHVSnapshotSpecList{
		SnapshotSpecs: []AHVSnapshotSpec{
			{VMUuid: vmUUID,
				SnapshotName: snapshotName},
		},
	}
	bodybytes, err := json.Marshal(vmspec)
	log.Debugf("Snapshot req: %s", string(bodybytes))
	req, err := http.NewRequest("POST", c.baseurl_ahv+"snapshots", bytes.NewBuffer(bodybytes))
	if err != nil {
		return "", err
	}

	err = c.do_request(req, func(body []byte) error {
		taskstruct := struct {
			TaskUUID string `json:"taskUuid"`
		}{}
		err := json.Unmarshal(body, &taskstruct)
		TaskUUID = taskstruct.TaskUUID
		return err
	})
	return
}

func (c *Client) DeleteVMSnapshotByUUID(UUID string) (TaskUUID string, err error) {
	req, _ := http.NewRequest("DELETE", c.baseurl_ahv+"snapshots/"+UUID, nil)

	err = c.do_request(req, func(body []byte) error {
		taskstruct := struct {
			TaskUUID string `json:"taskUuid"`
		}{}
		err := json.Unmarshal(body, &taskstruct)
		TaskUUID = taskstruct.TaskUUID
		return err
	})
	return
}

func (c *Client) GetSnapshotByUUID(UUID string) (*AHVSnapshotInfo, error) {
	req, _ := http.NewRequest("GET", c.baseurl_ahv+"snapshots/"+UUID, nil)

	var snap AHVSnapshotInfo
	err := c.do_request(req, func(body []byte) error {
		return json.Unmarshal(body, &snap)
	})

	return &snap, err
}

func (c *Client) GetContainerNameByUUID(UUID string) (string, error) {
	req, err := http.NewRequest("GET", c.baseurl_v1+"containers/"+UUID, nil)
	onlyname := struct {
		Name string `json:"name"`
	}{}
	err = c.do_request(req, func(body []byte) error {
		return json.Unmarshal(body, &onlyname)
	})
	return onlyname.Name, err
}

func (c *Client) GetTaskByUUID(UUID string) (*TaskInfo, error) {
	req, err := http.NewRequest("GET", c.baseurl_ahv+"tasks/"+UUID+"?includeEntityNames=true", nil)
	var task TaskInfo
	err = c.do_request(req, func(body []byte) error {
		return json.Unmarshal(body, &task)
	})
	return &task, err
}

func (c *Client) PollTaskForCompletion(UUID string) (*TaskInfo, error) {
	poll_period := 10 * time.Second
	var waited time.Duration
	log.Debugf("Polling task %s for completion", UUID)

	for {
		task, err := c.GetTaskByUUID(UUID)
		if err != nil {
			return nil, err
		}
		if task.ProgressStatus == "Failed" {
			return task, fmt.Errorf("%s: %s", task.MetaResponse.Error, task.MetaResponse.ErrorDetail)
		}
		if task.PercentageComplete == 100 {
			return task, err
		}
		log.Infof("Waiting for operation %s to complete. Progress %d. Taken so far %s", task.OperationType, task.PercentageComplete, waited)
		time.Sleep(poll_period)
		waited += poll_period
	}
}

type AHVImageSpec struct {
	Annotation      string             `json:"annotation"`
	ImageType       string             `json:"imageType"`
	Name            string             `json:"name"`
	ImageImportSpec AHVImageImportSpec `json:"imageImportSpec"`
}
type AHVImageImportSpec struct {
	ContainerUUID string `json:"containerUuid"`
	URL           string `json:"url"`
}

func (c *Client) CreateImageFromURL(name, annotation, container_uuid, url string) (*TaskInfo, error) {
	reqbody := fmt.Sprintf(`{"annotation":"%s",
	"imageType":"disk_image",
	"name":"%s",
	"imageImportSpec":{
		"containerUuid":"%s",
		"url":"%s"
	}
}`, annotation, name, container_uuid, url)
	req, err := http.NewRequest("POST", c.baseurl_ahv+"/images", bytes.NewBuffer([]byte(reqbody)))

	var task TaskInfo
	err = c.do_request(req, func(body []byte) error {
		return json.Unmarshal(body, &task)
	})

	return &task, err
}
