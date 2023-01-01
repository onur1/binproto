package binproto

// A Message represents a single binproto message.
//
// Each message starts with an header which is a varint encoded
// unsigned 64-bit integer which consists of an ID (first 60-bits) and
// a Channel number (last 4-bits), the rest of the message is payload.
type Message struct {
	ID      int
	Channel rune
	Data    []byte
}

// NewMessage returns a new Message.
func NewMessage(id int, ch rune, data []byte) *Message {
	return &Message{
		ID:      id,
		Channel: ch,
		Data:    data,
	}
}
