package alibaba

import (
	"bufio"
	"fmt"
	"github.com/denverdino/aliyungo/oss"
	"github.com/pkg/errors"
	"io"
	"k8s.io/klog/v2"
	"os"
	"strings"
)

func (n *Devel) CreateBucket(name string) error {
	if name == "" {
		return fmt.Errorf("empyt bucket name")
	}
	return n.OSS.Bucket(name).PutBucket(oss.Private)
}

func (n *Devel) GetObject(src string) ([]byte, error) {
	bName, mpath := n.Cfg.BucketName, src
	if strings.HasPrefix(src, "oss://") {
		segs := strings.Split(src, "/")
		if len(segs) < 4 {
			return nil, fmt.Errorf("invalid oss bucket: %s", src)
		}
		// override bucket name by user
		bName = segs[2]
		mpath = strings.Replace(src, fmt.Sprintf("oss://%s/", bName), "", -1)
	}
	klog.Infof("oss get object from [oss://%s/%s]", bName, mpath)
	bucket := n.OSS.Bucket(bName)
	data, err := bucket.Get(mpath)
	if err != nil {
		return nil, errors.Wrapf(err, "get oss object: path=[oss://%s/%s]", bName, mpath)
	}
	return data, nil
}

func (n *Devel) GetFile(src, dst string) error {
	bName, mpath := n.Cfg.BucketName, src
	if strings.HasPrefix(src, "oss://") {
		segs := strings.Split(src, "/")
		if len(segs) < 4 {
			return fmt.Errorf("invalid oss bucket: %s", src)
		}
		// override bucket name by user
		bName = segs[2]
		mpath = strings.Replace(src, fmt.Sprintf("oss://%s/", bName), "", -1)
	}
	klog.Infof("oss get file from [oss://%s/%s]", bName, mpath)
	bucket := n.OSS.Bucket(bName)
	reader, err := bucket.GetReader(mpath)
	if err != nil {
		return errors.Wrap(err, "get bucket reader")
	}
	defer reader.Close()
	desc, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		return errors.Wrapf(err, "open dest file:%s", dst)
	}
	defer desc.Close()
	cnt, err := io.Copy(bufio.NewWriterSize(desc, 10*1024*1024), reader)
	klog.Infof("get file[%s] from oss, read count[%d], to[%s]", src, cnt, dst)
	return err
}

func (n *Devel) PutFile(src, dst string) error {
	bName, mpath := n.Cfg.BucketName, dst
	if strings.HasPrefix(dst, "oss://") {
		segs := strings.Split(dst, "/")
		if len(segs) < 4 {
			return fmt.Errorf("invalid oss bucket: %s", dst)
		}
		// override bucket name by user
		bName = segs[2]
		mpath = strings.Replace(dst, fmt.Sprintf("oss://%s/", bName), "", -1)
	}
	klog.Infof("oss put file to [oss://%s/%s]", bName, mpath)

	bucket := n.OSS.Bucket(bName)
	desc, err := os.OpenFile(src, os.O_RDONLY, 0777)
	if err != nil {
		return errors.Wrapf(err, "open file: %s", src)
	}
	defer desc.Close()
	return bucket.PutFile(mpath, desc, oss.Private, oss.Options{})
}

func (n *Devel) PutObject(b []byte, dst string) error {
	bName, mpath := n.Cfg.BucketName, dst
	if strings.HasPrefix(dst, "oss://") {
		segs := strings.Split(dst, "/")
		if len(segs) < 4 {
			return fmt.Errorf("invalid oss bucket: %s", dst)
		}
		// override bucket name by user
		bName = segs[2]
		mpath = strings.Replace(dst, fmt.Sprintf("oss://%s/", bName), "", -1)
	}
	klog.Infof("oss put object to [oss://%s/%s]", bName, mpath)

	bucket := n.OSS.Bucket(bName)
	return bucket.Put(mpath, b, oss.DefaultContentType, oss.Private, oss.Options{})
}

func (n *Devel) DeleteObject(dst string) error {
	bName, mpath := n.Cfg.BucketName, dst
	if strings.HasPrefix(dst, "oss://") {
		segs := strings.Split(dst, "/")
		if len(segs) < 4 {
			return fmt.Errorf("invalid oss bucket: %s", dst)
		}
		// override bucket name by user
		bName = segs[2]
		mpath = strings.Replace(dst, fmt.Sprintf("oss://%s/", bName), "", -1)
	}
	klog.Infof("oss delete object [oss://%s/%s]", bName, mpath)
	bucket := n.OSS.Bucket(bName)
	return bucket.Del(mpath)
}
