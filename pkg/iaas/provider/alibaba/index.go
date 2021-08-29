package alibaba

import (
	"encoding/json"
	"fmt"
	api "github.com/aoxn/ooc/pkg/apis/alibabacloud.com/v1"
	"github.com/aoxn/ooc/pkg/utils"
	"github.com/denverdino/aliyungo/oss"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"strings"
)

func path(bucket, name string) string {
	return fmt.Sprintf("oss://%s/ovm/clusters/%s.json", bucket, name)
}

func (n *Devel) Save(id api.ClusterId) error {
	bName := n.Cfg.BucketName
	if bName == "" {
		return fmt.Errorf("oss bucket name should be provided in ooc config")
	}
	bucket := n.OSS.Bucket(bName)
	data := utils.PrettyJson(id)
	klog.Infof("trying to save cluster id to remote bucket: %s", id.Name)
	err := n.PutObject([]byte(data), path(bName, id.Name))
	if err == nil {
		return nil
	}
	klog.Errorf("put object: %s", err.Error())
	if strings.Contains(err.Error(), "NoSuchBucket") {
		err = bucket.PutBucket(oss.Private)
		if err != nil {
			return errors.Wrapf(err, "create bucket fail: %s", bName)
		}
		return n.PutObject([]byte(data), path(bName, id.Name))
	}
	return err
}
func (n *Devel) Get(id string) (api.ClusterId, error) {
	cid := api.ClusterId{}
	bName := n.Cfg.BucketName
	if bName == "" {
		return cid, fmt.Errorf("oss bucket name should be provided in ooc config")
	}
	data, err := n.GetObject(path(bName, id))
	if err != nil {
		return cid, errors.Wrapf(err, "get cluster: %s", id)
	}
	err = json.Unmarshal(data, &cid)
	if err != nil {
		return cid, errors.Wrapf(err, "unmarshal cluster: %s", id)
	}
	return cid, nil
}

func (n *Devel) Remove(id string) error {
	bName := n.Cfg.BucketName
	if bName == "" {
		return fmt.Errorf("oss bucket name should be provided in ooc config")
	}
	return n.DeleteObject(path(bName, id))
}

func (n *Devel) List(selector string) ([]api.ClusterId, error) {
	var cids []api.ClusterId
	bName := n.Cfg.BucketName
	if bName == "" {
		return cids, fmt.Errorf("oss bucket name should be provided in ooc config")
	}
	mlist, err := n.OSS.Bucket(bName).List("ovm/clusters", "", "", 1000)
	if err != nil {
		return cids, errors.Wrapf(err, "list object: %s", bName)
	}
	for _, v := range mlist.Contents {
		klog.Infof("get cluster: [%s]", v.Key)
		cid := api.ClusterId{}
		data, err := n.GetObject(v.Key)
		if err != nil {
			return cids, errors.Wrapf(err, "get object by key: %s", v.Key)
		}
		err = json.Unmarshal(data, &cid)
		if err != nil {
			return cids, errors.Wrapf(err, "unmarshal cluster: %s", v.Key)
		}
		cids = append(cids, cid)
	}
	return cids, nil
}
