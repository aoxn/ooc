package runtime

import (
	"github.com/aoxn/ooc/pkg/message"
	"k8s.io/klog/v2"
)

func NewDockerMovement() DockerMovement {
	return DockerMovement{Movement: message.NewMovement("runtime")}
}

type DockerMovement struct{ message.Movement }

func (d *DockerMovement) Run(bus *message.MessageBus) error {
	for {
		select {
		case msg := <-d.Queue():
			klog.Infof("Message received: %s", msg.From)
		}
	}
}
