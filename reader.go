package binproto

import (
	"errors"
	"io"
)

type Reader struct {
	rd       io.Reader
	r, w     int
	buf      []byte
	err      error
	state    int
	factor   uint64
	varint   uint64
	header   uint64
	length   int
	consumed int
	messages []*Message
	latest   []byte
	missing  int
}

const (
	minReadBufferSize = 16

	defaultMaxMessageSize = 8 * 1024 * 1024
	defaultBufSize        = 4096
)

var (
	ErrMessageSizeExceeded = errors.New("binproto: message too big")
	ErrMessageMalformed    = errors.New("binproto: message malformed")
)

var (
	errNegativeRead = errors.New("binproto: reader returned negative count from Read")
)

const maxConsecutiveEmptyReads = 16

func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultBufSize)
}

func NewReaderSize(rd io.Reader, size int) *Reader {
	if size < minReadBufferSize {
		size = minReadBufferSize
	}
	r := new(Reader)
	r.reset(make([]byte, size), rd)
	return r
}

func (b *Reader) fill() {
	// Slide existing data to beginning.
	if b.r > 0 {
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
		b.r = 0
	}

	length := len(b.buf)

	if b.w >= length {
		panic("bufio: tried to fill full buffer")
	}

	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		n, err := b.rd.Read(b.buf[b.w:])
		if n < 0 {
			panic(errNegativeRead)
		}
		b.w += n
		if err != nil {
			if errors.Is(err, io.EOF) && b.missing > 0 {
				err = io.ErrUnexpectedEOF
			}
			b.err = err
			return
		}

		if n > 0 {
			if length < n {
				b.err = io.ErrShortBuffer
				return
			}
			return
		}
	}
	b.err = io.ErrNoProgress
}

func (b *Reader) ReadMessage() (message *Message, err error) {
	bufferLength := len(b.buf)

	for {
		if b.err != nil {
			message = nil
			b.r = b.w
			err = b.readErr()
			break
		}

		// Found message?
		if len(b.messages) > 0 {
			message, b.messages = b.messages[0], b.messages[1:]
			b.missing = 0
			copy(b.buf, b.buf[b.r:b.w])
			b.w -= b.r
			b.r = 0
			break
		}

		// Reading ok?
		if b.state == 0 && b.r > 1 {
			b.r = b.w
			err = io.ErrNoProgress
			break
		}

		if b.r < b.w {
			if b.state == 2 {
				b.r = b.readMessage(b.buf[:b.w], b.r)
			} else {
				b.r = b.readVarint(b.buf[:b.w], b.r)
			}
			continue
		}

		if b.state == 2 && b.length == 0 {
			b.r = b.readMessage(b.buf, b.r)
			continue
		}

		// Is buffer big enough?
		remaining := b.length - b.consumed
		if b.w+remaining > bufferLength {
			b.r = b.w
			err = io.ErrShortBuffer
			break
		}

		b.fill()
	}

	return
}

func (b *Reader) Reset(r io.Reader) {
	if b.buf == nil {
		b.buf = make([]byte, defaultBufSize)
	}
	b.reset(b.buf, r)
}

func (b *Reader) next(data []byte, offset int) bool {
	switch b.state {
	case 0:
		b.state = 1
		b.factor = 1
		b.length = int(b.varint)
		b.consumed = 0
		b.varint = 0
		if b.length == 0 {
			b.state = 0
		}

		return true
	case 1:
		b.state = 2
		b.factor = 1
		b.header = b.varint
		b.length -= b.consumed
		b.consumed = 0
		b.varint = 0
		if b.length < 0 || b.length > defaultMaxMessageSize {
			b.destroy(ErrMessageSizeExceeded)

			return false
		}
		extra := len(data) - offset
		if b.length > extra {
			b.missing = b.length - extra
		}

		return true
	case 2:
		b.state = 0
		b.messages = append(b.messages, NewMessage(int(b.header>>4), rune(b.header&0b1111), b.latest))
		b.latest = nil

		return b.err == nil
	default:
		return false
	}
}

func (b *Reader) readMessage(data []byte, offset int) int {
	l := len(data)

	free := l - offset
	if free >= b.length {
		if b.latest != nil {
			copy(b.latest[len(b.latest)-b.length:], data[offset:])
		} else {
			b.latest = make([]byte, (offset+b.length)-offset)
			copy(b.latest, data[offset:offset+b.length])
		}

		offset += b.length

		if b.next(data, offset) {
			return offset
		}

		return l
	}

	if b.latest == nil {
		b.latest = make([]byte, b.length)
	}

	copy(b.latest[len(b.latest)-b.length:], data[offset:])

	b.length -= free

	return l
}

func (b *Reader) readVarint(data []byte, offset int) int {
	for ; offset < len(data); offset++ {
		b.varint += uint64(data[offset]&127) * b.factor
		b.consumed += 1

		if data[offset] < 128 {
			offset += 1

			if b.next(data, offset) {
				return offset
			}
			return len(data)
		}

		b.factor *= 128
	}

	if b.consumed >= 11 {
		b.destroy(ErrMessageMalformed)
	}

	return len(data)
}

func (b *Reader) readErr() error {
	err := b.err
	b.err = nil
	return err
}

func (b *Reader) reset(buf []byte, r io.Reader) {
	*b = Reader{
		rd:     r,
		buf:    buf,
		factor: 1,
	}
}

func (b *Reader) destroy(err error) {
	if err != nil {
		b.err = err
	}
}
