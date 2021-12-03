package rate

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"testing"
	"time"
)

func TestThrottle(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	rate := NewRateLimit("Describe", 10)
	err := rate.WaitCall(
		func() error {
			n := rand.Int31n(10)
			if n < 7 {
				klog.Errorf("api called with throttle possibility: %d, try again\n", n)
				return fmt.Errorf("throttle")
			}
			klog.Infof("api called succeed.%d\n", n)
			return nil
		},
	)
	if err != nil {
		klog.Errorf("wait call: %s\n", err.Error())
	}
}
