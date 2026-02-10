package messaging

type Consumer interface {
	Start(handler func(Message)) error
	Close() error
}
