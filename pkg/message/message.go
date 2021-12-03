package message

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"time"
)

var movements = make(map[string]Action)

func AddMovement(action Action) {
	_, ok := movements[action.Name()]
	if ok {
		panic(fmt.Sprintf("action was registered before: %s", action.Name()))
	}
	movements[action.Name()] = action
}

type MessageBus struct {
	Queue chan *Message
}

func (m *MessageBus) SendMessage(msg *Message) { m.Queue <- msg }

func (m *MessageBus) dispatch() {
	go func() {
		wait.Forever(
			func() {
				klog.Infof("start dispatch message")
				for {
					select {
					case msg := <-m.Queue:
						move := movements[msg.To.Name]
						if move == nil {
							klog.Infof("dispatch msg error: no movement found for %v. Message dropped\n", msg)
							continue
						}
						move.Queue() <- msg
					}
				}
			},
			1*time.Second,
		)
	}()
}

func (m *MessageBus) Run() error {
	for _, action := range movements {
		go func(act Action) {
			wait.Forever(
				func() {
					err := act.Run(m)
					if err != nil {
						klog.Infof("movement run error: %s", err.Error())
					}
				},
				1*time.Second,
			)
		}(action)
	}
	m.dispatch()
	return nil
}
