package messaging

type Message interface {
	ID() string
	Body() string
	Type() string
}
