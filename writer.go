package binproto

import (
	"bufio"
	"encoding/binary"
)

// A Writer implements convenience methods for writing
// requests or responses to a binary protocol network connection.
type Writer struct {
	wd *bufio.Writer
}

// NewWriter returns a new Writer writing to w.
func NewWriter(wd *bufio.Writer) *Writer {
	return &Writer{wd: wd}
}

// WriteMessage writes a variable number of messages to w.
func (w *Writer) WriteMessage(messages ...*Message) error {
	var err error
	if len(messages) == 1 {
		i := messages[0]
		_, err = w.wd.Write(send(i.ID, i.Channel, i.Data))
	} else {
		_, err = w.wd.Write(sendBatch(messages))
	}
	if err != nil {
		return err
	}

	if err := w.wd.Flush(); err != nil {
		return err
	}

	return nil
}

func send(id int, ch rune, data []byte) []byte {
	header := uint64(id)<<4 | uint64(ch)
	length := len(data) + encodingLength(header)
	payload := make([]byte, encodingLength(uint64(length))+length)

	n := binary.PutUvarint(payload, uint64(length))
	n += binary.PutUvarint(payload[n:], header)
	copy(payload[n:], data)

	return payload
}

func sendBatch(items []*Message) []byte {
	offset := 0

	var length int

	for _, v := range items {
		// 16 is >= the max size of the varints
		length += 16 + encodingLength(uint64(len(v.Data)))
	}

	payload := make([]byte, length)

	for _, v := range items {
		header := uint64(v.ID<<4) | uint64(v.Channel)
		l := uint64(len(v.Data) + encodingLength(header))

		offset += binary.PutUvarint(payload[offset:], l)
		offset += binary.PutUvarint(payload[offset:], header)

		copy(payload[offset:], v.Data)

		offset += len(v.Data)
	}

	return payload[0:offset]
}
