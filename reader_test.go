package binproto_test

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/onur1/binproto"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	desc   string
	reads  [][]byte
	expect []interface{}
	size   int
}

var maxID = int(math.Pow(2, 60)) - 1

func TestReadMessage(t *testing.T) {
	// Run test cases in normal and reversed order.
	test := func(prefix string, messages []interface{}, bytes [][]byte, bufferSize int) {
		for _, tt := range makeTestCases(prefix, bufferSize, messages, bytes, io.EOF) {
			tt := tt
			t.Run(tt.desc, func(t *testing.T) {
				runTest(t, tt)
			})
		}

		// Reverse.
		for i := len(messages)/2 - 1; i >= 0; i-- {
			j := len(messages) - 1 - i
			messages[i], messages[j] = messages[j], messages[i]
			bytes[i], bytes[j] = bytes[j], bytes[i]
		}

		for _, tt := range makeTestCases(prefix, bufferSize, messages, bytes, io.EOF) {
			tt := tt
			t.Run(tt.desc, func(t *testing.T) {
				runTest(t, tt)
			})
		}
	}

	// Send headers only.
	func() {
		var messages []interface{}
		var bytes [][]byte

		bufferSize := 16

		id := maxID
		for i := 1; id > 0; id = id / (i * 2) {
			i++
			messages = append(messages, newMessage(id, rune(i%15), 0))
			bytes = append(bytes, newBytes(id, rune(i%15), 0, 0, 0))
		}

		test("header only", messages, bytes, bufferSize)
	}()

	// Messages that fill up the entire buffer.
	func() {
		var messages []interface{}
		var bytes [][]byte

		bufferSize := 16

		id := maxID
		for i := 1; id > 0; id = id / (i * 2) {
			i++
			b := newBytes(id, rune(i%15), 0, 0, 0)
			messages = append(messages, newMessage(id, rune(i%15), bufferSize-len(b)))
			bytes = append(bytes, newBytes(id, rune(i%15), bufferSize-len(b), 0, 0))
		}

		test("with payload buffer full", messages, bytes, bufferSize)
	}()

	// Messages with smaller payloads.
	func() {
		var messages []interface{}
		var bytes [][]byte

		bufferSize := 16

		id := maxID
		for i := 1; id > 0; id = id / (i * 2) {
			i++
			b := newBytes(id, rune(i%15), 0, 0, 0)
			messages = append(messages, newMessage(id, rune(i%15), (bufferSize-len(b))/2))
			bytes = append(bytes, newBytes(id, rune(i%15), (bufferSize-len(b))/2, 0, 0))
		}

		test("with payload buffer half empty", messages, bytes, bufferSize)
	}()

	// Arbitrary tests.
	testCases := []testCase{
		{
			desc:   "basic",
			expect: expecting(newMessage(42, 3, 3), io.EOF),
			reads:  reading(newBytes(42, 3, 3, 0, 0)),
			size:   16,
		},
		{
			desc:   "smallest header",
			expect: expecting(newMessage(0, 0, 0), io.EOF),
			reads:  reading(newBytes(0, 0, 0, 0, 0)),
			size:   16,
		},
		{
			desc:   "smallest header with message",
			expect: expecting(newMessage(0, 0, 14), io.EOF),
			reads:  reading(newBytes(0, 0, 14, 0, 0)),
			size:   16,
		},
		{
			desc:   "large header",
			expect: expecting(newMessage(maxID, 15, 0), io.EOF),
			reads:  reading(newBytes(maxID, 15, 0, 0, 0)),
			size:   16,
		},
		{
			desc:   "large header with message",
			expect: expecting(newMessage(maxID, 15, 5), io.EOF),
			reads:  reading(newBytes(maxID, 15, 5, 0, 0)),
			size:   16,
		},
		{
			desc:   "receive chunked",
			expect: expecting(newMessage(0, 0, 2), io.EOF),
			reads: func() [][]byte {
				payload := newBytes(0, 0, 2, 0, 0)
				var xs [][]byte
				for i := 0; i < len(payload); i++ {
					xs = append(xs, payload[i:i+1])
				}
				return reading(xs...)
			}(),
			size: 16,
		},
		{
			desc:   "receive chunks in different sizes",
			expect: expecting(newMessage(42, 3, 130), io.EOF),
			reads: func() [][]byte {
				payload := newBytes(42, 3, 130, 0, 0)
				xs := [][]byte{
					payload[0:1],
					payload[1:3],
					payload[3:17],
					payload[17:56],
					payload[56:89],
					payload[89:107],
					payload[107:],
				}
				return reading(xs...)
			}(),
			size: 256,
		},
		{
			desc: "receive two in single read",
			expect: expecting(
				newMessage(5, 10, 4),
				newMessage(42, 3, 7),
				io.EOF,
			),
			reads: func() [][]byte {
				xs := [][]byte{
					newBytes(5, 10, 4, 0, 0),
					newBytes(42, 3, 7, 0, 0),
				}
				return reading(flattened(xs))
			}(),
			size: 16,
		},
		{
			desc: "receive two with small buffer",
			expect: expecting(
				io.ErrShortBuffer,
			),
			reads: func() [][]byte {
				xs := [][]byte{
					newBytes(5, 10, 5, 0, 0),
					newBytes(42, 3, 7, 0, 0),
				}
				return reading(flattened(xs))
			}(),
			size: 16,
		},
		{
			desc: "receive many in single read",
			expect: expecting(
				newMessage(5, 10, 2),
				newMessage(42, 3, 5),
				newMessage(27, 1, 8),
				newMessage(98993, 15, 100),
				io.EOF,
			),
			reads: func() [][]byte {
				return reading(flattened([][]byte{
					newBytes(5, 10, 2, 0, 0),
					newBytes(42, 3, 5, 0, 0),
					newBytes(27, 1, 8, 0, 0),
					newBytes(98993, 15, 100, 0, 0),
				}))
			}(),
			size: 256,
		},
		{
			desc: "receive many chunked",
			expect: expecting(
				newMessage(5, 10, 2),
				newMessage(42, 3, 5),
				newMessage(maxID, 1, 5),
				io.EOF,
			),
			reads: func() [][]byte {
				return reading(append(
					chunksOf(newBytes(5, 10, 2, 0, 0), 1),
					append(
						chunksOf(newBytes(42, 3, 5, 0, 0), 2),
						chunksOf(newBytes(maxID, 1, 5, 0, 0), 1)...,
					)...,
				)...)
			}(),
			size: 16,
		},
		{
			desc:   "empty write",
			expect: expecting(io.EOF),
		},
		{
			desc: "receive two with small buffer",
			expect: func() []interface{} {
				return expecting(
					newMessage(0, 1, 2),
					io.ErrShortBuffer,
				)
			}(),
			reads: func() [][]byte {
				b1 := newBytes(0, rune(1), 2, 0, 0)
				b2 := newBytes(42, rune(3), 14, 0, 0)

				return reading(b1, b2)
			}(),
			size: 16,
		},
		{
			desc: "receive two chunked with small buffer",
			expect: func() []interface{} {
				return expecting(
					newMessage(0, 1, 2),
					io.ErrShortBuffer,
				)
			}(),
			reads: func() [][]byte {
				b1 := newBytes(0, rune(1), 2, 0, 0)
				b2 := newBytes(42, rune(3), 14, 0, 0)
				b := append(b1, b2...)
				xs := chunksOf(b, 2)
				return reading(xs...)
			}(),
			size: 16,
		},
		{
			desc: "receive many chunked with small buffer",
			expect: expecting(
				newMessage(5, 10, 2),
				newMessage(42, 3, 5),
				io.ErrShortBuffer,
			),
			reads: func() [][]byte {
				return reading(append(
					chunksOf(newBytes(5, 10, 2, 0, 0), 1),
					append(
						chunksOf(newBytes(42, 3, 5, 0, 0), 2),
						chunksOf(newBytes(maxID, 1, 6, 0, 0), 1)...,
					)...,
				)...)
			}(),
			size: 16,
		},
		{
			desc:   "short buffer",
			expect: expecting(io.ErrShortBuffer),
			reads:  reading(newBytes(0, 0, 15, 0, 0)),
			size:   16,
		},
		{
			desc:   "the smallest message to raise no progress error",
			expect: expecting(io.ErrNoProgress),
			reads:  reading(make([]byte, 2)),
			size:   16,
		},
		{
			desc:   "the largest message to raise no progress error",
			expect: expecting(io.ErrNoProgress),
			reads:  reading(make([]byte, 16)),
			size:   16,
		},
		{
			desc:   "missing the last byte when closed",
			expect: expecting(io.ErrUnexpectedEOF),
			reads:  reading(newBytes(0, 0, 14, 0, 15)),
			size:   16,
		},
		{
			desc:   "missing bytes while buffer is full",
			expect: expecting(io.ErrShortBuffer),
			reads:  reading(newBytes(0, 0, 15, 0, 12)),
			size:   16,
		},
		{
			desc: "missing the last byte after many chunked",
			expect: expecting(
				newMessage(5, 10, 2),
				newMessage(42, 3, 5),
				io.ErrUnexpectedEOF,
			),
			reads: func() [][]byte {
				return reading(append(
					chunksOf(newBytes(5, 10, 2, 0, 0), 1),
					append(
						chunksOf(newBytes(42, 3, 5, 0, 0), 2),
						chunksOf(newBytes(0, 0, 14, 0, 15), 1)...,
					)...,
				)...)
			}(),
			size: 16,
		},
		{
			desc: "missing the last byte after single read",
			expect: expecting(
				newMessage(5, 10, 2),
				io.ErrUnexpectedEOF,
			),
			reads: reading(newBytes(5, 10, 2, 0, 0), newBytes(0, 0, 14, 0, 15)),
			size:  16,
		},
		{
			desc: "missing bytes after single read",
			expect: expecting(
				newMessage(5, 10, 2),
				io.ErrUnexpectedEOF,
			),
			reads: func() [][]byte {
				xs := reading(newBytes(5, 10, 2, 0, 0), newBytes(maxID, 0, 5, 0, 1))
				return xs
			}(),
			size: 16,
		},
		{
			desc: "missing bytes after single read with no payload",
			expect: expecting(
				newMessage(5, 10, 2),
				io.ErrUnexpectedEOF,
			),
			reads: func() [][]byte {
				xs := reading(newBytes(5, 10, 2, 0, 0), newBytes(0, 0, 0, 0, 1))
				return xs
			}(),
			size: 16,
		},
		{
			desc: "max consecutive reads reached",
			expect: expecting(
				newMessage(5, 10, 2),
				io.ErrNoProgress,
			),
			reads: func() [][]byte {
				xs := chunksOf(newBytes(5, 10, 2, 0, 0), 2)
				for i := 0; i < 100; i++ {
					xs = append(xs, make([]byte, 0))
				}
				return xs
			}(),
			size: 16,
		},
		{
			desc: "receive big messages chunked",
			expect: expecting(
				newMessage(42, 3, 1e5),
				newMessage(maxID, 1, 1e5),
				io.EOF,
			),
			reads: func() [][]byte {
				xs := reading(append(
					chunksOf(newBytes(42, 3, 1e5, 0, 0), 2048),
					chunksOf(newBytes(maxID, 1, 1e5, 0, 0), 3052)...,
				)...)
				return xs
			}(),
			size: 1e5 + 4096,
		},
	}
	start := 0
	for _, tt := range testCases[start:] {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			runTest(t, tt)
		})
	}
}

func BenchmarkReadMessage(b *testing.B) {
	b.ReportAllocs()

	var buf bytes.Buffer
	br := bufio.NewReader(&buf)
	r := binproto.NewReaderSize(br, 256)
	data := []byte(fill(56))

	for i := 0; i < b.N; i++ {
		id := i % maxID
		ch := rune(i % 16)

		buf.Write(send(id, ch, data))
		m, err := r.ReadMessage()
		if err != nil {
			b.Fatal(err)
		}
		if m.ID != id || m.Channel != ch || string(m.Data) != string(data) {
			b.Fatal(fmt.Sprintf("expected: %d %d %s, got: %d %d %s", id, ch, string(data), m.ID, m.Channel, string(m.Data)))
		}
	}
}

type testReader struct {
	q [][]byte
}

func (r *testReader) Read(p []byte) (n int, err error) {
	if len(r.q) < 1 {
		return 0, io.EOF
	}
	b := r.q[0]
	r.q = r.q[1:]
	copy(p, b)
	return len(b), nil
}

func reader(q [][]byte, size int) *binproto.Reader {
	return binproto.NewReaderSize(&testReader{q: q}, size)
}

func newMessage(id int, ch rune, l int) *binproto.Message {
	return &binproto.Message{
		ID:      id,
		Channel: ch,
		Data:    []byte(fill(l)),
	}
}

func newBytes(id int, ch rune, l int, start, end int) []byte {
	out := send(id, ch, []byte(fill(l)))
	if end == 0 {
		return out[start:]
	} else {
		return out[start:end]
	}
}

func fill(times int) string {
	s := ""
	chars := "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < times; i++ {
		pos := i % len(chars)
		s += chars[pos : pos+1]
	}
	return s
}

func runTest(t *testing.T, tt testCase) {
	r := reader(tt.reads, tt.size)
	if len(tt.expect) < 1 {
		panic("must expect a result")
	}
	for _, v := range tt.expect {
		res, err := r.ReadMessage()
		if expectedErr, ok := v.(error); ok {
			assert.EqualError(t, err, expectedErr.Error(), tt.desc)
		} else {
			assert.Nil(t, err)
			if expectedMessage, ok := v.(*binproto.Message); ok {
				assert.EqualValues(t, expectedMessage, res, tt.desc)
			} else {
				t.Fail()
			}
		}
	}
}

func makeTestCases(prefix string, bufferSize int, messages []interface{}, bytes [][]byte, expectedErr error) (testCases []testCase) {
	for i := range messages {
		testCases = append(testCases, testCase{
			desc:   fmt.Sprintf("%s single read %d", prefix, i),
			expect: expecting(messages[i], expectedErr),
			reads:  reading(bytes[i]),
			size:   bufferSize,
		})
	}

	expected := expecting(append(messages, expectedErr)...)

	testCases = append(testCases, testCase{
		desc:   fmt.Sprintf("%s multiple reads", prefix),
		expect: expected,
		reads:  bytes,
		size:   bufferSize,
	})

	flattenedBytes := flattened(bytes)

	for i := 1; i < 12; i++ {
		testCases = append(testCases, testCase{
			desc:   fmt.Sprintf("%s multiple reads chunked %d", prefix, i),
			expect: expected,
			reads:  chunksOf(flattenedBytes, i),
			size:   bufferSize,
		})
	}

	testCases = append(testCases, testCase{
		desc:   fmt.Sprintf("%s single read many messages", prefix),
		expect: expected,
		reads:  reading(flattenedBytes),
		size:   len(flattenedBytes),
	})

	testCases = append(testCases, testCase{
		desc:   fmt.Sprintf("%s randomly chunked", prefix),
		expect: expected,
		reads:  randomChunksOf(flattenedBytes, 1, 11),
		size:   bufferSize,
	})

	return
}

func flattened(xs [][]byte) []byte {
	if len(xs) < 1 {
		panic("invalid argument")
	}
	ret := xs[0]
	if len(xs) == 1 {
		return ret
	}
	for _, x := range xs[1:] {
		ret = append(ret, x...)
	}
	return ret
}

func chunksOf(xs []byte, chunkSize int) [][]byte {
	if len(xs) == 0 || chunkSize == 0 {
		panic("invalid argument")
	}
	divided := make([][]byte, (len(xs)+chunkSize-1)/chunkSize)
	prev := 0
	i := 0
	till := len(xs) - chunkSize
	for prev < till {
		next := prev + chunkSize
		divided[i] = xs[prev:next]
		prev = next
		i++
	}
	divided[i] = xs[prev:]
	return divided
}

func randomChunksOf(b []byte, min, max int) [][]byte {
	if min > len(b) || max <= min || max > len(b) {
		panic("invalid args")
	}
	var out [][]byte
	for len(b) > 0 {
		random := int(math.Min(float64(rand.Intn(max-min)+min), float64(len(b))))
		out = append(out, b[0:random])
		b = b[random:]
	}
	return out
}

func reading(xs ...[]byte) [][]byte {
	return xs
}

func expecting(xs ...interface{}) []interface{} {
	return xs
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	n1 = uint64(math.Pow(2, 7))
	n2 = uint64(math.Pow(2, 14))
	n3 = uint64(math.Pow(2, 21))
	n4 = uint64(math.Pow(2, 28))
	n5 = uint64(math.Pow(2, 35))
	n6 = uint64(math.Pow(2, 42))
	n7 = uint64(math.Pow(2, 49))
	n8 = uint64(math.Pow(2, 56))
	n9 = uint64(math.Pow(2, 63))
)

func encodingLength(i uint64) int {
	if i < n1 {
		return 1
	} else if i < n2 {
		return 2
	} else if i < n3 {
		return 3
	} else if i < n4 {
		return 4
	} else if i < n5 {
		return 5
	} else if i < n6 {
		return 6
	} else if i < n7 {
		return 7
	} else if i < n8 {
		return 8
	} else if i < n9 {
		return 9
	}
	return 10
}

func send(id int, ch rune, data []byte) []byte {
	if id > maxID {
		panic("binproto: ID too big")
	}
	header := uint64(id)<<4 | uint64(ch)
	length := len(data) + encodingLength(header)
	payload := make([]byte, encodingLength(uint64(length))+length)
	n := binary.PutUvarint(payload, uint64(length))
	n += binary.PutUvarint(payload[n:], header)
	copy(payload[n:], data)
	return payload
}
