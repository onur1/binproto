package binproto

import "math"

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
