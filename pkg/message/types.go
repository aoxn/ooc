package message

//
// Message driven system
//

type Payload struct {
	Context map[string]interface{}
}

type Address struct {
	Name string

	Route  Route
	Detail string
}

type Route struct {
	// Scope Local|System
	Scope string

	// Target can be complicated address
	Target Target
}

type Target struct {
}

type Message struct {
	//
	Id string

	// From where the message coming from
	From Address

	// To to whom the message should be sent.
	To Address

	// Quote this is a reply message to last.
	Quote *Message

	// Payload
	Payload *Payload
}

func New() *Message {
	return &Message{}
}
