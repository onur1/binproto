package binproto_test

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/onur1/binproto"
)

func TestSend(t *testing.T) {
	var buf bytes.Buffer
	w := binproto.NewWriter(bufio.NewWriter(&buf))
	err := w.WriteMessage(newMessage(42, 3, 2))
	if s := buf.String(); s != "\x04\xa3\x05ab" || err != nil {
		t.Fatalf("s=%q; err=%s", s, err)
	}
}

func TestSendBatch(t *testing.T) {
	var buf bytes.Buffer
	w := binproto.NewWriter(bufio.NewWriter(&buf))
	msg := newMessage(42, 3, 2)
	err := w.WriteMessage(msg, msg)
	if s := buf.String(); s != "\x04\xa3\x05ab\x04\xa3\x05ab" || err != nil {
		t.Fatalf("s=%q; err=%s", s, err)
	}
}
