package message

import (
	"fmt"
)

type Action interface {
	Name() string

	Queue() chan *Message

	Run(bus *MessageBus) error
}

type Movement struct {
	name  string
	queue chan *Message
}

func (m *Movement) Name() string { return m.name }

func (m *Movement) Queue() chan *Message { return m.queue }

func (m *Movement) Run(bus *MessageBus) error { return fmt.Errorf("unimplemented") }

func NewMovement(name string) Movement {
	return Movement{name: name, queue: make(chan *Message, 20)}
}
