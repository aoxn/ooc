package rate

import (
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type QuotaM struct {
	// QuotaKey  quota name
	QuotaKey string

	// QuotaQPS  quota value
	QuotaQPS float32
}

type Throttle struct {
	Lock    *sync.RWMutex
	Limiter map[string]*RateLimit
}

var F = Throttle{}

func AddLimiter(key string, qps float32) {
	F.Lock.Lock()
	defer F.Lock.Unlock()
	F.Limiter[key] = NewRateLimit(key, qps)
}

func GetLimiter(key string) *RateLimit {
	limit, ok := F.Limiter[key]
	if !ok {
		return nil
	}
	return limit
}

func Call(key string, call Callable) error {
	ratec := GetLimiter(key)
	if ratec == nil {
		return call()
	}
	return ratec.WaitCall(call)
}

type Callable func() error

var (
	PLUNK_UP   float32 = 1.2
	PLUNK_DOWN float32 = 0.83
)

func NewRateLimit(key string, qps float32) *RateLimit {

	return &RateLimit{
		Quota: &QuotaM{
			QuotaKey: key,
			QuotaQPS: qps,
		},
		Concurrency: 100,
	}
}

type RateLimit struct {
	Lock        *sync.RWMutex
	Quota       *QuotaM
	Concurrency float32
}

func (r *RateLimit) WaitCall(call Callable) error {
	// lock for concurrent execute.
	// it is meaningless to execute concurrently
	r.Lock.Lock()
	defer r.Lock.Unlock()
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-t.C:
			rand.Seed(time.Now().UnixNano())
			p := float32(rand.Intn(10)) / 10
			klog.Infof("tick 1s: %f, %f\n", p, r.possibility())
			if p < r.possibility() {
				// hit, get the chance to run
				err := call()
				if err != nil {
					if !strings.Contains(err.Error(), "throttle") {
						// report an error and out
						return err
					}
					// we are throttled, adjust possibility
					if r.Concurrency < 1000 {
						// up concurrency
						r.Concurrency = r.Concurrency * PLUNK_UP
					}
					klog.Infof("Evaluated: %f\n", r.Concurrency)
					continue
				}
				if r.Concurrency > 1 {
					// down concurrency
					r.Concurrency = r.Concurrency * PLUNK_DOWN
				}
				klog.Infof("call success.\n")
				return nil
			}
		}
	}
}

func (r *RateLimit) possibility() float32 {
	// it is ok for possibility > 1
	return r.Quota.QuotaQPS / r.Concurrency * 2.0
}
