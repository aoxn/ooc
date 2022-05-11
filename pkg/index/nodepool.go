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

func nPath(bucket, id, name string) string {
	return fmt.Sprintf("oss://%s/wdrip/nodepools/%s/%s.json", bucket, id, name)
}

func NewNodePoolIndex(
	cid string, store pd.ObjectStorage,
) *NodePoolIndex {
	return &NodePoolIndex{cid: cid, store: store}
}

type NodePoolIndex struct {
	cid   string
	store pd.ObjectStorage
}

func (n *NodePoolIndex) SaveNodePool(id api.NodePool) error {
	bName := n.store.BucketName()
	if bName == "" {
		return fmt.Errorf("oss bucket name should be provided in wdrip config")
	}

	data := utils.PrettyJson(id)
	klog.Infof("trying to save NodePoolIndex id to remote bucket: %s", id.Name)
	err := n.store.PutObject([]byte(data), nPath(bName, n.cid, id.Name))
	if err == nil {
		return nil
	}
	klog.Errorf("put object: %s", err.Error())
	if strings.Contains(err.Error(), "NoSuchBucket") {
		err = n.store.EnsureBucket(bName)
		if err != nil {
			return errors.Wrapf(err, "create bucket fail: %s", bName)
		}
		return n.store.PutObject([]byte(data), nPath(bName, n.cid, id.Name))
	}
	return err
}
func (n *NodePoolIndex) GetNodePool(id string) (api.NodePool, error) {
	cid := api.NodePool{}
	bName := n.store.BucketName()
	if bName == "" {
		return cid, fmt.Errorf("oss bucket name should be provided in wdrip config")
	}
	data, err := n.store.GetObject(nPath(bName, n.cid, id))
	if err != nil {
		return cid, errors.Wrapf(err, "get NodePoolIndex: %s", id)
	}
	err = json.Unmarshal(data, &cid)
	if err != nil {
		return cid, errors.Wrapf(err, "unmarshal NodePoolIndex: %s", id)
	}
	return cid, nil
}

func (n *NodePoolIndex) RemoveNodePool(id string) error {
	bName := n.store.BucketName()
	if bName == "" {
		return fmt.Errorf("oss bucket name should be provided in wdrip config")
	}
	return n.store.DeleteObject(nPath(bName, n.cid, id))
}

func (n *NodePoolIndex) ListNodePools(selector string) ([]api.NodePool, error) {
	var cids []api.NodePool
	bName := n.store.BucketName()
	if bName == "" {
		return cids, fmt.Errorf("oss bucket name should be provided in wdrip config")
	}
	mlist, err := n.store.ListObject(fmt.Sprintf("wdrip/nodepools/%s", n.cid))
	if err != nil {
		return cids, errors.Wrapf(err, "list clusters: %s", bName)
	}
	for _, v := range mlist {
		cid := api.NodePool{}
		err = json.Unmarshal(v, &cid)
		if err != nil {
			return cids, errors.Wrapf(err, "unmarshal NodePoolIndex")
		}
		cids = append(cids, cid)
	}
	return cids, nil
}
