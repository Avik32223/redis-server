package transport

type Message struct {
	Peer    Peer
	Payload []byte
}

type Peer interface {
	Close() error
	Send([]byte) error
}

type Transport interface {
	Addr() string
	Listen() error
	Consume() <-chan Message
	Close() error
}
