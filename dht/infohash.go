package dht

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"math/rand"
	"time"
)

const ihLength = 20

// Infohash -
type Infohash []byte

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

// FromString -
func (ih *Infohash) FromString(s string) error {
	switch len(s) {
	case 20:
		// Byte string
		*ih = Infohash([]byte(s))
		return nil
	case 40:
		b, err := hex.DecodeString(s)
		if err != nil {
			return err
		}
		*ih = Infohash(b)
	}
	return nil
}

func (ih Infohash) GenNeighbour(other Infohash) Infohash {
	s := append(ih[:10], other[10:]...)
	return Infohash(s)
}

func randomInfoHash() (ih Infohash) {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash := sha1.New()
	io.WriteString(hash, time.Now().String())
	io.WriteString(hash, string(random.Int()))
	return Infohash(hash.Sum(nil))
}
