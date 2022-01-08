package binproto_test

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/onur1/binproto"
)

func ExampleDial() {
	s := &server{}

	time.AfterFunc(time.Millisecond*1, func() {
		c, err := binproto.Dial("tcp", ":4242")
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			for {
				msg, err := c.ReadMessage()
				if err != nil {
					log.Fatal(err)
					return
				}

				fmt.Printf("%d %d %s\n", msg.ID, msg.Channel, msg.Data)

				s.close()
			}
		}()

		_, err = c.Send(binproto.NewMessage(42, 3, []byte("hi")))
		if err != nil {
			log.Fatal(err)
		}
	})

	if err := s.serve("tcp", ":4242"); err != nil {
		log.Fatal(err)
	}

	// output:
	// 42 3 hi
	// 112 5 hey
}

type server struct {
	listener net.Listener
}

func (s *server) handle(conn net.Conn) {
	defer conn.Close()

	c := binproto.NewConn(conn)

	for {
		msg, err := c.ReadMessage()
		if err != nil {
			fmt.Printf("error: %v", err)
			return
		}

		fmt.Printf("%d %d %s\n", msg.ID, msg.Channel, msg.Data)

		_, err = c.Send(binproto.NewMessage(112, 5, []byte("hey")))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (s *server) serve(network, address string) error {
	l, err := net.Listen(network, address)
	if err != nil {
		return err
	}

	s.listener = l

	for {
		if s.listener == nil {
			break
		}

		c, err := l.Accept()
		if err != nil {
			continue
		}

		go s.handle(c)
	}

	return nil
}

func (s *server) close() error {
	err := s.listener.Close()
	s.listener = nil
	return err
}
