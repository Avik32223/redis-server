package transport

type HandshakeFunc func(Peer) error

func NoOpHandshake(Peer) error { return nil }
