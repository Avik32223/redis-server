package transport

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

type TCPPeer struct {
	net.Conn
}

func (t *TCPPeer) Close() error {
	if t.Conn != nil {
		t.Conn.Close()
	}
	// fmt.Printf("tcp: closed connection. %#v \n", t)
	return nil
}

func (t *TCPPeer) Send(b []byte) error {
	_, err := t.Conn.Write(b)
	return err
}

type Receiver func(io.Reader) ([]byte, error)

func DefaultReceiver(c io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(c)
	scanner.Split(bufio.ScanLines)

	if scanner.Scan() {
		b := scanner.Bytes()
		return b, nil
	}

	if err := scanner.Err(); err != nil {
		b := scanner.Bytes()
		return b, err
	}

	return nil, io.EOF
}

// TCPTransport implements Transport
type TCPTransport struct {
	listener     net.Listener
	listenerAddr string
	consumeCh    chan Message

	Receive   Receiver
	Handshake HandshakeFunc
}

func NewTCPTransport(addr string) *TCPTransport {
	return &TCPTransport{
		listenerAddr: addr,
		consumeCh:    make(chan Message),
		Receive:      DefaultReceiver,
		Handshake:    NoOpHandshake,
	}
}

func (t *TCPTransport) Addr() string {
	return t.listenerAddr
}

func (t *TCPTransport) Consume() <-chan Message {
	return t.consumeCh
}

func (t *TCPTransport) Listen() error {
	var err error
	t.listener, err = net.Listen("tcp", t.listenerAddr)
	if err != nil {
		return err
	}

	go t.startListening()
	return nil
}

func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) startListening() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("tcp: error. %s\n", err)
		}

		go t.handleConnection(conn)
	}
}

func (t *TCPTransport) handleConnection(c net.Conn) {
	peer := TCPPeer{Conn: c}
	defer peer.Close()
	// fmt.Printf("tcp: new connection. %#v \n", peer)

	if err := t.Handshake(&peer); err != nil {
		peer.Send([]byte(err.Error()))
		return
	}

	for {
		b, err := t.Receive(c)
		if err != nil {
			if err == io.EOF {
				return
			}
			continue
		}
		msg := Message{
			Peer:    &peer,
			Payload: b,
		}
		t.consumeCh <- msg

	}
}
