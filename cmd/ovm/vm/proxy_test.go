package vm

import (
	"net"
	"sync"
	"testing"
	"time"
)

func TestA(t *testing.T)  {

	conn, err := net.Dial("tcp", "127.0.0.1:8888")
	if err != nil {
		t.Fatalf("error : %s", err)
	}
	grp := sync.WaitGroup{}
	grp.Add(1)
	go func() {
		bb := make([]byte,1024)
		n, err := conn.Read(bb)
		if err != nil {
			t.Logf("read error: %s", err.Error())
		}
		t.Logf("read %d byte: xxx", n)
		grp.Done()
	}()
	t.Logf("sleep 5s")
	time.Sleep(5*time.Second)

	conn.Close()

	t.Logf("sleep 3")
	time.Sleep(3*time.Second)
	t.Logf("wait")
	grp.Wait()
	t.Logf("finishe")
}