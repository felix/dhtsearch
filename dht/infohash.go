package dht

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"time"
)

const ihLength = 20

// Infohash is a 160 bit (20 byte) value
type Infohash []byte

// InfohashFromString converts a 40 digit hexadecimal string to an Infohash
func InfohashFromString(s string) (*Infohash, error) {
	switch len(s) {
	case 20:
		// Binary string
		ih := Infohash([]byte(s))
		return &ih, nil
	case 40:
		// Hex string
		b, err := hex.DecodeString(s)
		if err != nil {
			return nil, err
		}
		ih := Infohash(b)
		return &ih, nil
	default:
		return nil, fmt.Errorf("invalid length %d", len(s))
	}
}

func (ih Infohash) String() string {
	return hex.EncodeToString(ih)
}
func (ih Infohash) Valid() bool {
	// TODO
	return len(ih) == 20
}

func (ih Infohash) Equal(other Infohash) bool {
	if len(ih) != len(other) {
		return false
	}
	for i := 0; i < len(ih); i++ {
		if ih[i] != other[i] {
			return false
		}
	}
	return true
}

// Distance determines the distance to another infohash as an integer
func (ih Infohash) Distance(other Infohash) int {
	i := 0
	for ; i < 20; i++ {
		if ih[i] != other[i] {
			break
		}
	}

	if i == 20 {
		return 160
	}

	xor := ih[i] ^ other[i]

	j := 0
	for (xor & 0x80) == 0 {
		xor <<= 1
		j++
	}
	return 8*i + j
}

func generateNeighbour(first, second Infohash) Infohash {
	s := append(second[:10], first[10:]...)
	return Infohash(s)
}

func randomInfoHash() (ih Infohash) {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash := sha1.New()
	io.WriteString(hash, time.Now().String())
	io.WriteString(hash, string(random.Int()))
	return Infohash(hash.Sum(nil))
}
