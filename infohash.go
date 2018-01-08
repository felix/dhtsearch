package dhtsearch

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"math/rand"
	"time"
)

const ihLength = 20

func genInfoHash() string {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash := sha1.New()
	io.WriteString(hash, time.Now().String())
	io.WriteString(hash, string(random.Int()))
	ih := hash.Sum(nil)
	return string(ih)
}

func genNeighbour(first, second string) string {
	s := second[:10] + first[10:]
	return s
}

func decodeInfoHash(in string) (b string, err error) {
	var h []byte
	h, err = hex.DecodeString(in)
	if len(h) != ihLength {
		return "", errors.New("invalid length")
	}
	return string(h), err
}

func isValidInfoHash(id string) bool {
	ih, err := hex.DecodeString(id)
	if err != nil {
		return false
	}
	return len(ih) == ihLength
}
