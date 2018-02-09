package dht

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
)

const transIDBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func newTransactionID() string {
	b := make([]byte, 2)
	for i := range b {
		b[i] = transIDBytes[rand.Int63()%int64(len(transIDBytes))]
	}
	return string(b)
}

// makeQuery returns a query-formed data.
func makeQuery(t, q string, a map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"t": t,
		"y": "q",
		"q": q,
		"a": a,
	}
}

// makeResponse returns a response-formed data.
func makeResponse(t string, r map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"t": t,
		"y": "r",
		"r": r,
	}
}

// parseMessage parses the basic data received from udp.
// It returns a map value.
func parseMessage(data interface{}) (map[string]interface{}, error) {
	response, ok := data.(map[string]interface{})
	if !ok {
		return nil, errors.New("response is not dict")
	}

	if err := checkKeys(response, [][]string{{"t", "string"}, {"y", "string"}}); err != nil {
		return nil, err
	}

	return response, nil
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
func compactNodeInfoToString(cni string) string {
	if len(cni) == 6 {
		return fmt.Sprintf("%d.%d.%d.%d:%d", cni[0], cni[1], cni[2], cni[3], (uint16(cni[4])<<8)|uint16(cni[5]))
	} else if len(cni) == 18 {
		b := []byte(cni[:16])
		return fmt.Sprintf("[%s]:%d", net.IP.String(b), (uint16(cni[16])<<8)|uint16(cni[17]))
	} else {
		return ""
	}
}
