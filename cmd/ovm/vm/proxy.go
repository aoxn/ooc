package vm

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"net"
	"os"
	"sync"
	"time"
)

type flagProxy struct {
	publishes []string
}

func NewProxyCommand() *cobra.Command {
	flags := flagProxy{}
	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "proxy local port to unix socket",
		Long:  "unknown",
		RunE: func(cmd *cobra.Command, args []string) error {
			// start do proxy
			doProxy(&flags)
			return nil
		},
	}
	cmd.Flags().StringArrayVarP(&flags.publishes, "publish", "p", []string{}, "publish port")
	return cmd
}

func doProxy(p *flagProxy) {
	if len(p.publishes) <=0 {
		klog.Infof("proxy publish address should be provided with -p")
		os.Exit(1)
	}
	pub := NewPublish(p.publishes[0])
	proxy := func() {
		pub.SetStreamHandler(NewStreamUnix(pub.rnetwork, pub.raddr))
		err := pub.Listen()
		if err != nil {
			klog.Errorf("proxy error with: %s", err.Error())
		}
	}
	wait.Forever(proxy, 5 * time.Second)

}

func NewStreamUnix(network,addr string) func(conn net.Conn) error{
	proxy := func (conn net.Conn) error{
		defer closeConn(conn)
		vzconn, err := net.Dial(network, addr)
		if err != nil {
			return fmt.Errorf("error dialing remote addr: %s", err)
		}
		grp := &sync.WaitGroup{}
		grp.Add(1)
		defer closeConn(vzconn)
		go streamCopy(vzconn, conn, grp)
		go streamCopy(conn, vzconn, nil)
		grp.Wait()
		return nil
	}
	return proxy
}
