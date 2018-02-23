package bencode

import (
	"errors"
	"fmt"
	"strconv"
)

// Decode decodes a bencoded string to string, int, list or map.
func Decode(data []byte) (r interface{}, err error) {
	r, _, err = decodeItem(data, 0)
	return r, err
}

// DecodeString decodes a string from a given index
// It returns the string, the number of bytes successfully read
func DecodeString(data []byte, start int) (r string, n int, err error) {
	if start >= len(data) || data[start] < '0' || data[start] > '9' {
		err = errors.New("bencode: invalid string length")
		return r, 1, err
	}

	prefix, i, err := readUntil(data, start, ':')
	end := start + i
	if err != nil {
		return r, end - start, err
	}

	length, err := strconv.ParseInt(string(prefix), 10, 0)
	if err != nil {
		return r, end - start, err
	}
	end = end + int(length)

	if end > len(data) || end < i {
		err = errors.New("bencode: string length out of range")
		return r, end - start, err
	}

	return string(data[start+i : end]), end - start, nil
}

// DecodeInt decodes an integer value
// It returns the integer and the number of bytes successfully read
func DecodeInt(data []byte, start int) (r int64, end int, err error) {
	if start >= len(data) || data[start] != 'i' {
		err = errors.New("bencode: invalid integer")
		return r, end, err
	}

	prefix, n, err := readUntil(data, start, 'e')
	if err != nil {
		return r, n, err
	}

	r, err = strconv.ParseInt(string(prefix[1:]), 10, 64)
	return r, n, err
}

// DecodeList decodes a list value
// It returns the array and the number of bytes successfully read
func DecodeList(data []byte, start int) (r []interface{}, end int, err error) {
	if start >= len(data) {
		return r, end, errors.New("bencode: list range error")
	}
	if data[start] != 'l' {
		return r, end, errors.New("bencode: invalid list")
	}

	end = start + 1

	// Empty list
	if data[end] == 'e' {
		return r, 2, nil
	}

	var item interface{}

	var n int
	for end < len(data) {
		item, n, err = decodeItem(data, end)
		end = end + n
		if err != nil {
			return r, end - start, err
		}
		r = append(r, item)

		if data[end] == 'e' {
			return r, end - start + 1, nil
		}
	}

	return r, end, errors.New("bencode: invalid list termination")
}

// DecodeDict decodes a dict as a map
// It returns the map and the position of the last character
func DecodeDict(data []byte, start int) (map[string]interface{}, int, error) {
	r := make(map[string]interface{})

	if start >= len(data) {
		return r, 0, errors.New("bencode: dict range error")
	}

	if data[start] != 'd' {
		return r, 1, errors.New("bencode: invalid dict")
	}

	end := start + 1

	// Empty dict
	if data[end] == 'e' {
		return r, 2, nil
	}

	for end < len(data) {
		key, n, err := DecodeString(data, end)
		end = end + n
		if err != nil {
			return r, end, errors.New("bencode: invalid dict key")
		}

		if end >= len(data) {
			return r, end, errors.New("bencode: dict range error")
		}

		item, n, err := decodeItem(data, end)
		end = end + n

		if err != nil {
			return r, end, err
		}

		r[key] = item

		if data[end] == 'e' {
			return r, end - start + 1, nil
		}
	}
	return r, end, errors.New("bencode: invalid dict termination")
}

// decodeItem decodes the next type at the given index
func decodeItem(data []byte, start int) (r interface{}, n int, err error) {
	switch data[start] {
	case 'l':
		return DecodeList(data, start)
	case 'd':
		return DecodeDict(data, start)
	case 'i':
		return DecodeInt(data, start)
	default:
		return DecodeString(data, start)
	}
}

// Read until the given character
// Returns the slice before the character and the number of bytes successfully read
func readUntil(data []byte, start int, c byte) ([]byte, int, error) {
	i := start
	for ; i < len(data); i++ {
		if data[i] == c {
			return data[start:i], i - start + 1, nil
		}
	}
	return data, i - start, fmt.Errorf("bencode: '%b' not found", c)
}
