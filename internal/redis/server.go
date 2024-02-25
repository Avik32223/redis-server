package redis

import (
	"bufio"
	"fmt"
	"io"

	"github.com/Avik32223/redis-server/internal/transport"
)

type servermode string

const standalone servermode = "standalone"

type Server struct {
	id        string
	mode      servermode
	Transport transport.Transport
	quitCh    chan struct{}

	state State
}

func NewServer(addr string) *Server {
	t := transport.NewTCPTransport(addr)
	t.Receive = receive

	s := Server{
		id:        "default",
		mode:      standalone,
		Transport: t,
		state:     NewState(),
	}
	return &s
}

func receive(i io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(i)

	// The custom split method helps us accumulate buffered incomming data and split them by valid msessages.
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// Redis clients always communicate by sending commands as an arry of bulk strings.
		// Hoping to find a valid array from the 0th index
		if rune(data[0]) == '*' {
			c := eatArray(data, 0)

			// if a valid redis client message is found,
			// we split the data into a valid token and return.
			// This helps is pipelining commands (data contains multiple commands).
			if c > 0 {
				return c, data[:c], nil
			}
			// This handles the case when we couldn't find
			// a valid array of bulk strings (presumably from a redis client).
			// We will still need to validate the whole message later
			// as redis supports inline communication.
			// We return the complete message to be processed later.
			if atEOF {
				return 0, data, bufio.ErrFinalToken
			}

			// Request more data
			return 0, nil, nil

		} else {
			// If the start of a message does not hint of a redis command.
			// We'll assume its an inline command
			return bufio.ScanLines(data, atEOF)
		}
	})

	if scanner.Scan() {
		b := scanner.Bytes()
		return b, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nil, io.EOF
}

func (s *Server) Start() error {
	fmt.Printf("Starting redis server on %s.\n", s.Transport.Addr())
	if err := s.Transport.Listen(); err != nil {
		return err
	}
	defer s.Transport.Close()
	for {
		select {
		case msg := <-s.Transport.Consume():
			s.HandleMessage(msg)

		case <-s.quitCh:
			return nil
		}
	}
}

func (s *Server) Stop() error {
	close(s.quitCh)
	return nil
}

func (s *Server) HandleMessage(m transport.Message) error {
	x, err := RunCommand(s.state, m.Payload)
	if err != nil {
		x, _ := Serialize(err, nil)
		return m.Peer.Send([]byte(x))
	}
	res, err := Serialize(x, nil)
	if err != nil {
		x, _ := Serialize(err, nil)
		return m.Peer.Send([]byte(x))
	}
	return m.Peer.Send([]byte(res))
}
