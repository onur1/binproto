package binproto

import (
	"bufio"
	"io"
	"net"
	"net/textproto"
)

// A Conn represents a binary network protocol connection.
// It consists of a Reader and Writer to manage I/O
// and a Pipeline (which is borrowed from textproto) to sequence
// concurrent requests on the connection.
type Conn struct {
	Reader
	Writer
	textproto.Pipeline
	conn io.ReadWriteCloser
}

// NewConn returns a new Conn using conn for I/O.
func NewConn(conn io.ReadWriteCloser) *Conn {
	r, w := NewReaderSize(bufio.NewReader(conn), 16), NewWriter(bufio.NewWriter(conn))
	return &Conn{
		Reader: *r,
		Writer: *w,
		conn:   conn,
	}
}

// Send is a convenience method that sends a variable number of messages
// after waiting its turn in the pipeline.
// Send returns the id of the command, for use with StartResponse and EndResponse.
func (c *Conn) Send(m ...*Message) (id uint, err error) {
	id = c.Next()
	c.StartRequest(id)
	err = c.WriteMessage(m...)
	c.EndRequest(id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// Dial connects to the given address on the given network using net.Dial
// and then returns a new Conn for the connection.
func Dial(network, addr string) (*Conn, error) {
	c, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	return NewConn(c), nil
}
