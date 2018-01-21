package bencode

import (
	"errors"
	"fmt"
	"sort"
)

// Encode encodes a string, int, dict or list value to a bencoded string.
func Encode(data interface{}) ([]byte, error) {
	return encodeItem(data)
}

// EncodeString encodes a string value.
func EncodeString(data string) ([]byte, error) {
	length := fmt.Sprintf("%d:", len(data))
	out := make([]byte, 0, len(length)+len(data)+1)
	out = append(out, []byte(length)...)
	return append(out, []byte(data)...), nil
}

// EncodeInt encodes a int value.
func EncodeInt(data int64) ([]byte, error) {
	ib := fmt.Sprintf("i%de", data)
	return []byte(ib), nil
}

// EncodeDict encodes a dict value.
func EncodeDict(data map[string]interface{}) ([]byte, error) {
	out := make([]byte, 0, 2)
	out = append(out, 'd')

	// Sort keys
	list := make(sort.StringSlice, len(data))
	i := 0
	for key := range data {
		list[i] = key
		i++
	}
	list.Sort()

	for _, key := range list {
		keyb, err := EncodeString(key)
		if err != nil {
			return nil, err
		}
		value, err := encodeItem(data[key])
		if err != nil {
			return nil, err
		}
		out = append(out, keyb...)
		out = append(out, value...)
	}
	return append(out, 'e'), nil
}

// EncodeList encodes a list value.
func EncodeList(data []interface{}) ([]byte, error) {
	out := make([]byte, 0, 2)
	out = append(out, 'l')

	for _, item := range data {
		b, err := encodeItem(item)
		if err != nil {
			return nil, err
		}
		out = append(out, b...)
	}
	return append(out, 'e'), nil
}

// EncodeItem encodes an item of dict or list.
func encodeItem(data interface{}) ([]byte, error) {
	switch v := data.(type) {
	case string:
		return EncodeString(v)
	case int:
		return EncodeInt(int64(v))
	case int16:
		return EncodeInt(int64(v))
	case int32:
		return EncodeInt(int64(v))
	case int64:
		return EncodeInt(int64(v))
	case []interface{}:
		return EncodeList(v)
	case map[string]interface{}:
		return EncodeDict(v)
	default:
		return nil, errors.New("bencode: invalid type to encode")
	}
}
