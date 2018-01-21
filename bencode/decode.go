package bencode

import (
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"
)

// Decode decodes a bencoded string to string, int, list or map.
func Decode(data []byte) (r interface{}, err error) {
	r, _, err = decodeItem(data, 0)
	return r, err
}

// DecodeString decodes a string from a given index
// It returns the string and the position of the last character
func DecodeString(data []byte, start int) (r string, end int, err error) {
	if start >= len(data) || data[start] < '0' || data[start] > '9' {
		err = errors.New("bencode: invalid string length")
		return r, end, err
	}

	prefix, i, err := readUntil(data, start, ':')
	if err != nil {
		return r, end, err
	}

	length, err := strconv.ParseInt(string(prefix), 10, 0)
	if err != nil {
		return r, end, err
	}

	end = i + int(length)

	if end > len(data) || end < i {
		err = errors.New("bencode: string length out of range")
		return r, end, err
	}

	return string(data[i:end]), end - 1, nil
}

// DecodeInt decodes an integer value
// It returns the integer and the position of the last character
func DecodeInt(data []byte, start int) (r int64, end int, err error) {
	if start >= len(data) || data[start] != 'i' {
		err = errors.New("bencode: invalid integer")
		return r, end, err
	}

	prefix, end, err := readUntil(data, start, 'e')
	if err != nil {
		return r, end, err
	}

	r, err = strconv.ParseInt(string(prefix[1:]), 10, 64)
	return r, end - 1, err
}

// DecodeList decodes a list value
// It returns the array and the position of the last character
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
		return r, end, nil
	}

	var item interface{}

	for end < len(data) {
		item, end, err = decodeItem(data, end)
		if err != nil {
			return r, end, err
		}
		r = append(r, item)

		end++
		char, _ := utf8.DecodeRune(data[end:])
		if char == 'e' {
			return r, end, nil
		}
	}

	return r, end, errors.New("bencode: invalid list termination")
}

// DecodeDict decodes a dict as a map
// It returns the map and the position of the last character
func DecodeDict(data []byte, start int) (r map[string]interface{}, end int, err error) {
	if start >= len(data) {
		return r, end, errors.New("bencode: dict range error")
	}

	if data[start] != 'd' {
		return r, end, errors.New("bencode: invalid dict")
	}

	end = start + 1

	// Empty dict
	if data[end] == 'e' {
		return r, end, nil
	}

	var item interface{}
	var key string
	r = make(map[string]interface{})

	for end < len(data) {
		key, end, err = DecodeString(data, end)
		if err != nil {
			return r, end, errors.New("bencode: invalid dict key")
		}

		if end >= len(data) {
			return r, end, errors.New("bencode: dict range error")
		}

		end++
		item, end, err = decodeItem(data, end)
		if err != nil {
			return r, end, err
		}

		r[key] = item

		end++
		char, _ := utf8.DecodeRune(data[end:])
		if char == 'e' {
			return r, end, nil
		}
	}
	return r, end, errors.New("bencode: invalid dict termination")
}

// decodeItem decodes the next type at the given index
// It returns the index of the last character decoded
func decodeItem(data []byte, start int) (r interface{}, end int, err error) {

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
// Returns the slice before the character and the index of the next character
func readUntil(data []byte, start int, c byte) ([]byte, int, error) {
	i := start
	for ; i < len(data); i++ {
		if data[i] == c {
			return data[start:i], i + 1, nil
		}
	}
	return data, i, fmt.Errorf("bencode: '%b' not found", c)
}
