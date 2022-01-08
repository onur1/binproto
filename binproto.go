package binproto

import (
	"bufio"
	"io"
	"net"
	"net/textproto"
)

type Conn struct {
	Reader
	Writer
	textproto.Pipeline
	conn io.ReadWriteCloser
}

func NewConn(conn io.ReadWriteCloser) *Conn {
	r, w := NewReaderSize(bufio.NewReader(conn), 16), NewWriter(bufio.NewWriter(conn))
	return &Conn{
		Reader: *r,
		Writer: *w,
		conn:   conn,
	}
}

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

func (c *Conn) Close() error {
	return c.conn.Close()
}

func Dial(network, addr string) (*Conn, error) {
	c, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	return NewConn(c), nil
}
