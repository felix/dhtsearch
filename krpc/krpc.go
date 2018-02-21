package krpc

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
)

const transIDBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func NewTransactionID() string {
	b := make([]byte, 2)
	for i := range b {
		b[i] = transIDBytes[rand.Int63()%int64(len(transIDBytes))]
	}
	return string(b)
}

// makeQuery returns a query-formed data.
func MakeQuery(transaction, query string, data map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"t": transaction,
		"y": "q",
		"q": query,
		"a": data,
	}
}

// makeResponse returns a response-formed data.
func MakeResponse(transaction string, data map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"t": transaction,
		"y": "r",
		"r": data,
	}
}

func GetString(data map[string]interface{}, key string) (string, error) {
	val, ok := data[key]
	if !ok {
		return "", fmt.Errorf("krpc: missing key %s", key)
	}
	out, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("krpc: key type mismatch")
	}
	return out, nil
}

func GetInt(data map[string]interface{}, key string) (int, error) {
	val, ok := data[key]
	if !ok {
		return 0, fmt.Errorf("krpc: missing key %s", key)
	}
	out, ok := val.(int)
	if !ok {
		return 0, fmt.Errorf("krpc: key type mismatch")
	}
	return out, nil
}

func GetMap(data map[string]interface{}, key string) (map[string]interface{}, error) {
	val, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("krpc: missing key %s", key)
	}
	out, ok := val.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("krpc: key type mismatch")
	}
	return out, nil
}

func GetList(data map[string]interface{}, key string) ([]interface{}, error) {
	val, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("krpc: missing key %s", key)
	}
	out, ok := val.([]interface{})
	if !ok {
		return nil, fmt.Errorf("krpc: key type mismatch")
	}
	return out, nil
}

// parseKeys parses keys. It just wraps parseKey.
func checkKeys(data map[string]interface{}, pairs [][]string) (err error) {
	for _, args := range pairs {
		key, t := args[0], args[1]
		if err = checkKey(data, key, t); err != nil {
			break
		}
	}
	return err
}

// parseKey parses the key in dict data. `t` is type of the keyed value.
// It's one of "int", "string", "map", "list".
func checkKey(data map[string]interface{}, key string, t string) error {
	val, ok := data[key]
	if !ok {
		return fmt.Errorf("krpc: missing key %s", key)
	}

	switch t {
	case "string":
		_, ok = val.(string)
	case "int":
		_, ok = val.(int)
	case "map":
		_, ok = val.(map[string]interface{})
	case "list":
		_, ok = val.([]interface{})
	default:
		return errors.New("krpc: invalid type")
	}

	if !ok {
		return errors.New("krpc: key type mismatch")
	}

	return nil
}

// Swiped from nictuku
func DecodeCompactNodeAddr(cni string) string {
	if len(cni) == 6 {
		return fmt.Sprintf("%d.%d.%d.%d:%d", cni[0], cni[1], cni[2], cni[3], (uint16(cni[4])<<8)|uint16(cni[5]))
	} else if len(cni) == 18 {
		b := []byte(cni[:16])
		return fmt.Sprintf("[%s]:%d", net.IP.String(b), (uint16(cni[16])<<8)|uint16(cni[17]))
	} else {
		return ""
	}
}

func EncodeCompactNodeAddr(addr string) string {
	var a []uint8
	host, port, _ := net.SplitHostPort(addr)
	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}
	aa, _ := strconv.ParseUint(port, 10, 16)
	c := uint16(aa)
	if ip2 := net.IP.To4(ip); ip2 != nil {
		a = make([]byte, net.IPv4len+2, net.IPv4len+2)
		copy(a, ip2[0:net.IPv4len]) // ignore bytes IPv6 bytes if it's IPv4.
		a[4] = byte(c >> 8)
		a[5] = byte(c)
	} else {
		a = make([]byte, net.IPv6len+2, net.IPv6len+2)
		copy(a, ip)
		a[16] = byte(c >> 8)
		a[17] = byte(c)
	}
	return string(a)
}

func int2bytes(val int64) []byte {
	data, j := make([]byte, 8), -1
	for i := 0; i < 8; i++ {
		shift := uint64((7 - i) * 8)
		data[i] = byte((val & (0xff << shift)) >> shift)

		if j == -1 && data[i] != 0 {
			j = i
		}
	}

	if j != -1 {
		return data[j:]
	}
	return data[:1]
}
