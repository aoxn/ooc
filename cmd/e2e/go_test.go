package e2e

import (
	"fmt"
	"github.com/aoxn/ooc/pkg/iaas/provider"
	"math"
	"runtime"
	"testing"
	"time"
)

func bar() {
	fmt.Println("bar:", runtime.NumGoroutine())
}

func foo() {
	fmt.Println("foo:", runtime.NumGoroutine())
	go bar()
	time.Sleep(100 * time.Millisecond)
}

func TestEPM(t *testing.T) {
	fmt.Println("NumGoroutine:", runtime.NumGoroutine())
	fmt.Println("NumCPU:      ", runtime.NumCPU())

	fmt.Println("NumMax:   ", runtime.GOMAXPROCS(2))

	time.Sleep(100 * time.Millisecond)
}

func TestDuration(t *testing.T) {
	const (
		base   = 100 * time.Millisecond
		max    = 5 * time.Second
		factor = 2
	)
	duration := base
	for {
		if err := returnError(); err != nil {
			fmt.Printf("start - %d %v\n", duration, err)
			// exponential backoff
			time.Sleep(duration)
			fmt.Printf("after -%d %v\n", duration, err)

			duration = time.Duration(math.Min(float64(max), factor*float64(duration)))
			fmt.Printf("mmmmm - %d %v\n", duration, err)

			continue
		}
		// reset backoff if we have a success
		duration = base

		fmt.Println("finish")
	}
}

func returnError() error {
	return fmt.Errorf("error hahsh")
}


func TestClojour(t *testing.T) {
	var id provider.Id
	id = provider.Id{
		Name: "XXXXX",
	}

	defer func(id *provider.Id) {
		fmt.Printf("Defer: %x, %v\n",id, *id)
	}(&id)

	id = provider.Id{
		Name:  "YYYYYY",
	}
	fmt.Printf("normal: %v\n", id)
}
