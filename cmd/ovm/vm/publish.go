package vm

import (
	"fmt"
	"github.com/mdlayher/vsock"
	"github.com/pkg/errors"
	"io"
	"k8s.io/klog/v2"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

const (
	P_VSOCK = "vsock"
	P_TCP   = "tcp"
	P_UNIX  = "unix"
)

func NewPublish(addr string) *Publish {

	// unix@/tmp/unix.sock:tcp@8880
	// tcp@80:tcp@90
	// 80:90
	// tcp@80:90
	// 80:tcp@9000
	// 999
	breakDown := func(add string) (string,string){
		port := strings.Split(add, "@")
		if len(port) <= 1 {
			return "tcp", port[0]
		}
		// network, addr
		return port[0], port[1]
	}
	addrs := strings.Split(addr, ":")
	if len(addrs) <= 1 {
		network, port := breakDown(addrs[0])
		return &Publish{
			laddr: port,
			raddr: port,
			lnetwork: network,
			rnetwork: network,
		}
	}

	lnetwork, lport := breakDown(addrs[0])

	rnetwork, rport := breakDown(addrs[1])
	klog.Infof("%s,     %s, %s,   %s", lnetwork,lport, rnetwork, rport)
	return &Publish{
		laddr: lport,
		raddr: rport,
		lnetwork: lnetwork,
		rnetwork: rnetwork,
	}
}

type Publish struct {
	// tcp & udp
	lnetwork string
	laddr    string

	raddr    string
	rnetwork string

	streamHandler    func(conn net.Conn) error
}

func Port(sp string) uint32 {
	port,err := strconv.Atoi(sp)
	if err != nil {
		panic(fmt.Sprintf("convert port: %s, %s", sp, err.Error()))
	}
	return uint32(port)
}

func (p *Publish) String() string {
	return fmt.Sprintf("%s@%s:%s@%s", p.lnetwork,p.laddr, p.rnetwork,p.raddr)
}

func (p *Publish) SetStreamHandler(handler func(net.Conn)error) { p.streamHandler = handler }

func (p *Publish) Listen() error {
	var (
		err error
		lt  net.Listener
	)
	if p.lnetwork == "unix" {
		err := ensuresock(p.laddr)
		if err != nil {
			return errors.Wrapf(err, "ensure unix socket path %s", p)
		}
	}
	switch p.lnetwork {
	case P_VSOCK:
		lt, err = vsock.Listen(Port(p.laddr))
	case P_TCP, P_UNIX:
		lt, err = net.Listen(p.lnetwork, p.laddr)
	}
	if err != nil {
		return errors.Wrapf(err,"proxy listen, %s: %v", p.laddr, err)
	}
	klog.Infof("publish address: %s", p)
	for {
		conn, err := lt.Accept()
		if err != nil {
			klog.Infof("error accepting connection", err)
			continue
		}
		proxy := func(con net.Conn) {
			defer func() {
				if r := recover(); r != nil {
					klog.Errorf("recover failed")
				}
			}()
			err := p.streamHandler(con)
			if err != nil {
				klog.Errorf("run proxy: %s", err.Error())
			}
		}
		go proxy(conn)
	}
}

func streamCopy(dst io.Writer, src io.ReadCloser,grp *sync.WaitGroup) {
	n, err := io.Copy(dst, src)
	if err != nil {
		klog.Errorf("stream copy: %d bytes copied, %s", n, err.Error())
	}
	if grp != nil {
		grp.Done()
	}
	klog.Infof("stream copy finish: %x -> %x, %d bytes copied", &dst, &src, n)
}



func closeConn(con io.Closer) {
	err := con.Close()
	if err != nil {
		klog.Errorf("close conn: %s", err.Error())
	}
	//klog.Infof("close conn")
}


func ensuresock(path string) error {
	spath := filepath.Dir(path)
	err := os.MkdirAll(spath, 0700)
	if err != nil {
		return errors.Wrapf(err, "ensure unix socket path %s", path)
	}

	err = syscall.Unlink(path)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "unlink unix socket file %s", path)
	}
	return nil
}