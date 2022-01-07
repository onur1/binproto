package binproto

type Message struct {
	ID      int
	Channel rune
	Data    []byte
}

func NewMessage(id int, ch rune, data []byte) *Message {
	return &Message{
		ID:      id,
		Channel: ch,
		Data:    data,
	}
}
