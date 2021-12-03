package dev

import (
	"bufio"
	"fmt"
	"github.com/denverdino/aliyungo/oss"
	"github.com/pkg/errors"
	"io"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"strings"
)

func (n *Devel) GetObject(src string) ([]byte, error) {
	if !strings.HasPrefix(src,"oss://") {
		return nil, fmt.Errorf("source path should be a oss bucket, %s", src)
	}
	segs := strings.Split(src,"/")
	if len(segs) < 4 {
		return nil, fmt.Errorf("invalid oss bucket: %s", src)
	}
	bName := segs[2]
	path := strings.Replace(src, fmt.Sprintf("oss://%s/",bName), "",-1)
	bucket := n.OSS.Bucket(bName)
	data,err := bucket.Get(path)
	if err != nil {
		return nil, errors.Wrap(err,"get oss object")
	}
	return data, nil
}

func (n *Devel) GetFile(src, dst string) error {
	if !strings.HasPrefix(src,"oss://") {
		return fmt.Errorf("source path should be a oss bucket, %s", src)
	}
	segs := strings.Split(src,"/")
	if len(segs) < 4 {
		return fmt.Errorf("invalid oss bucket: %s", src)
	}
	bName := segs[2]
	path := strings.Replace(src, fmt.Sprintf("oss://%s/",bName), "",-1)
	bucket := n.OSS.Bucket(bName)
	reader,err := bucket.GetReader(path)
	if err != nil {
		return errors.Wrap(err, "get bucket reader")
	}
	defer reader.Close()
	desc,err := os.OpenFile(dst,os.O_RDWR|os.O_CREATE,0777)
	if err != nil {
		return errors.Wrapf(err,"open dest file:%s", dst)
	}
	defer desc.Close()
	cnt, err := io.Copy(bufio.NewWriterSize(desc, 10*1024*1024), reader)
	klog.Infof("get file[%s] from oss, read count[%d], to[%s]",src,cnt,dst)
	return err
}

func (n *Devel) PutFile(src, dst string) error {
	if !strings.HasPrefix(dst,"oss://") {
		return fmt.Errorf("dst path should be a oss bucket, %s", dst)
	}
	segs := strings.Split(dst,"/")
	if len(segs) < 4 {
		return fmt.Errorf("invalid oss bucket: %s", dst)
	}
	bName := segs[2]
	bucket := n.OSS.Bucket(bName)
	desc, err := os.OpenFile(src,os.O_RDONLY, 0777)
	if err != nil {
		return errors.Wrapf(err,"open file: %s", src)
	}
	defer desc.Close()
	mdst := filepath.Join(segs[3:]...)
	return bucket.PutFile(mdst,desc,oss.Private, oss.Options{})
}

func (n *Devel) PutObject(b []byte, dst string) error {
	if !strings.HasPrefix(dst,"oss://") {
		return fmt.Errorf("dst path should be a oss bucket, %s", dst)
	}
	segs := strings.Split(dst,"/")
	if len(segs) < 4 {
		return fmt.Errorf("invalid oss bucket: %s", dst)
	}
	bName := segs[2]
	bucket := n.OSS.Bucket(bName)
	mdst := filepath.Join(segs[3:]...)
	return bucket.Put(mdst,b,oss.DefaultContentType, oss.Private,oss.Options{})
}

func (n *Devel) DeleteObject(f string) error {
	if !strings.HasPrefix(f,"oss://") {
		return fmt.Errorf("dst del path should be a oss bucket, %s", f)
	}
	segs := strings.Split(f,"/")
	if len(segs) < 4 {
		return fmt.Errorf("invalid oss bucket: %s", f)
	}
	bName := segs[2]
	bucket := n.OSS.Bucket(bName)
	return bucket.Del(filepath.Join(segs[3:]...))
}