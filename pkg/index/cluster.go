package index

import (
	"encoding/json"
	"fmt"
	api "github.com/aoxn/wdrip/pkg/apis/alibabacloud.com/v1"
	pd "github.com/aoxn/wdrip/pkg/iaas/provider"
	"github.com/aoxn/wdrip/pkg/utils"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
	"strings"
)

func path(bucket, name string) string {
	return fmt.Sprintf("oss://%s/wdrip/clusters/%s.json", bucket, name)
}

func NewClusterIndex(
	id string, store pd.ObjectStorage,
) *ClusterIndex {
	return &ClusterIndex{id: id, store: store}
}

type ClusterIndex struct {
	id    string
	store pd.ObjectStorage
}

func (n *ClusterIndex) SaveCluster(id api.ClusterId) error {
	bName := n.store.BucketName()
	if bName == "" {
		return fmt.Errorf("oss bucket name should be provided in wdrip config")
	}

	data := utils.PrettyJson(id)
	klog.Infof("trying to save ClusterIndex id to remote bucket: %s", id.Name)
	err := n.store.PutObject([]byte(data), path(bName, id.Name))
	if err == nil {
		return nil
	}
	klog.Errorf("put object: %s", err.Error())
	if strings.Contains(err.Error(), "NoSuchBucket") {
		err = n.store.EnsureBucket(bName)
		if err != nil {
			return errors.Wrapf(err, "create bucket fail: %s", bName)
		}
		return n.store.PutObject([]byte(data), path(bName, id.Name))
	}
	return err
}
func (n *ClusterIndex) GetCluster(id string) (api.ClusterId, error) {
	cid := api.ClusterId{}
	bName := n.store.BucketName()
	if bName == "" {
		return cid, fmt.Errorf("oss bucket name should be provided in wdrip config")
	}
	data, err := n.store.GetObject(path(bName, n.id))
	if err != nil {
		return cid, errors.Wrapf(err, "get ClusterIndex: %s", n.id)
	}
	err = json.Unmarshal(data, &cid)
	if err != nil {
		return cid, errors.Wrapf(err, "unmarshal ClusterIndex: %s", n.id)
	}
	return cid, nil
}

func (n *ClusterIndex) RemoveCluster(id string) error {
	bName := n.store.BucketName()
	if bName == "" {
		return fmt.Errorf("oss bucket name should be provided in wdrip config")
	}
	return n.store.DeleteObject(path(bName, n.id))
}

func (n *ClusterIndex) ListCluster(selector string) ([]api.ClusterId, error) {
	var cids []api.ClusterId
	bName := n.store.BucketName()
	if bName == "" {
		return cids, fmt.Errorf("oss bucket name should be provided in wdrip config")
	}
	mlist, err := n.store.ListObject("wdrip/clusters")
	if err != nil {
		return cids, errors.Wrapf(err, "list clusters: %s", bName)
	}
	for _, v := range mlist {
		cid := api.ClusterId{}
		err = json.Unmarshal(v, &cid)
		if err != nil {
			return cids, errors.Wrapf(err, "unmarshal ClusterIndex")
		}
		cids = append(cids, cid)
	}
	return cids, nil
}
