package dhtsearch

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
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
		return "", fmt.Errorf("Decoded infoHash is incorrect length, got %d", len(h))
	}
	return string(h), err
}

/*
func isValidInfoHash(id string) bool {
	ih, err := hex.DecodeString(id)
	if err != nil {
		return false
	}
	return len(ih) == ihLength
}

func (ih infoHash) xor(other infoHash) (ret []byte) {
	if len(ih) != len(other) {
		return []byte("")
	}
	ret = make([]byte, ihLength)
	for i := 0; i < ihLength; i++ {
		ret[i] = ih[i] ^ other[i]
	}
	return
}

// Effectively the "distance"
// XORed then number of common bits
func (ih infoHash) prefixLen(other infoHash) (ret int) {
	//fmt.Printf("ih = %s, other = %s\n", ih.asHex(), other.asHex())
	id1, id2 := []byte(ih), []byte(other)

	xor := make([]byte, ihLength)
	i := 0
	for ; i < ihLength; i++ {
		xor[i] = id1[i] ^ id2[i]
	}

	for i := 0; i < ihLength; i++ {
		for j := 0; j < 8; j++ {
			if (xor[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}
	return ihLength*8 - 1
}

// Comparitor for iterable
func (ih infoHash) Less(other interface{}) bool {
	for i := 0; i < ihLength; i++ {
		if ih[i] != other.(infoHash)[i] {
			return ih[i] < other.(infoHash)[i]
		}
	}
	return false
}
*/
