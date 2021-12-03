package monit

import (
	h "github.com/aoxn/ovm/pkg/operator/controllers/help"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/klog/v2"
	"time"
)

type Action func() error

type Check interface {
	Check() (bool, error)
	Name()  string
	Limit() flowcontrol.RateLimiter
	Threshold() int
}

type BaseCheck struct {
}

func (m *BaseCheck) Name() string { return "base.check" }

func (m *BaseCheck) Check() (bool, error) { return true, nil }

func (m *BaseCheck) Threshold() int { return 6 }

func (m *BaseCheck) Action() error { return nil }

func NewThreshedCheck(check Check) *ThreshedCheck {
	return &ThreshedCheck{check:check}
}

type ThreshedCheck struct {
	check Check
	threshed   int
	lastFailed string
	lastGood   string
	lastReset  string
}

func (t *ThreshedCheck) Reset() { t.threshed = 0 }

func (t *ThreshedCheck) ThreshMeet() bool {
	return t.threshed > t.check.Threshold()
}

func NewMonitor() *Monitor{
	return &Monitor{checks: map[string]*ThreshedCheck{},lastRun: time.Now()}
}

type Monitor struct {
	checks  map[string]*ThreshedCheck
	actions []Action
	lastRun time.Time
}

func (m *Monitor) WithCheck(check ...Check) {
	for _, v := range check {
		_, ok := m.checks[v.Name()]
		if ok {
			klog.Warningf("duplicated register checker: %s", v.Name())
		}
		m.checks[v.Name()] = NewThreshedCheck(v)
	}
}

func (m *Monitor) WithAction(action ...Action) {
	m.actions = append(m.actions, action...)
}

func (m *Monitor) StartMonit() error {

	// 1. check
	// 2. threshold
	// 3. call iaas.recover . record last recover call time

	meetAll := func() bool {
		for _, v := range m.checks {
			if !v.ThreshMeet() {
				return false
			}
		}
		return true
	}
	check := func() {
		if meetAll() {
			klog.Infof("time.now=%s, last.run=%s",
				time.Now().Format("2006-01-02 15:04:05"),
				m.lastRun.Format("2006-01-02 15:04:05"),
			)

			if !time.Now().After(m.lastRun.Add(5 * time.Minute)) {
				klog.Infof("wait for 5 minutes after last run")
			}else {
				klog.Infof("all threshold reached, run top level actions")
				for k, action := range m.actions {
					err := action()
					if err != nil {
						klog.Errorf("run action[%d] failed: %s",k, err.Error())
					}
				}
				m.lastRun = time.Now()
				klog.Infof("run action finished")
			}
		}
		m.CheckRound()
	}
	wait.Forever(check, 10 * time.Second)
	panic("unreachable code")
}

func (m *Monitor) CheckRound() {
	for k, v := range m.checks {
		if ! v.check.Limit().TryAccept() {
			klog.Infof("not accept for throttle")
			continue
		}

		klog.Infof("accept for throttle")
		ok, err := v.check.Check()
		if err != nil {
			klog.Errorf( "check[%s] check fail: %s", k, err.Error())
			continue
		}
		if !ok {
			v.threshed += 1
			v.lastFailed = h.Now()
			klog.Warningf("[AbnormalCase] ++threshold.  current=%d", v.threshed)
		}else {
			// immediately reset threshold once check return ok
			// for careful reason.
			v.Reset()
			v.lastGood  = h.Now()
			v.lastReset = h.Now()
			klog.Infof("[CaseNormal] return normal, reset counter")
		}
	}
}
