//go:build darwin
// +build darwin

package vm

import (
	"fmt"
	"github.com/aoxn/ovm/pkg/utils/vz"
	"k8s.io/klog/v2"
	"net"
	"sync"
)

func NewStreamVSOCK(
	vso *vz.VirtioSocketDevice, port uint32,
) func(conn net.Conn) error {

	proxy := func(conn net.Conn) error {
		var ferr error
		klog.Infof("[host] prepare to connect to vm port=%d", port)

		onConnect := func(vzconn *vz.VirtioSocketConnection, err error) {
			if err != nil {
				ferr = err
				return
			}
			defer closeConn(conn)
			defer closeConn(vzconn)
			klog.Infof("[host]connect to vm success %d",port)

			grp := &sync.WaitGroup{}
			grp.Add(1)
			go streamCopy(vzconn, conn,grp)
			go streamCopy(conn, vzconn,nil)

			grp.Wait()
			klog.Infof("[host] stream copy finished [%d]",port)
		}
		if vso == nil {
			return fmt.Errorf("[host]empty socket device, [%v]", vso)
		}

		vso.ConnectToPort(port, onConnect)
		return ferr
	}
	return proxy
}
