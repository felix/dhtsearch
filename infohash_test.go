package dhtsearch

import (
	"testing"
)

var hashes = []struct {
	s     string
	valid bool
}{
	{"59066769b9ad42da2e508611c33d7c4480b3857b", true},
	{"59066769b9ad42da2e508611c33d7c4480b3857", false},
	{"59066769b9ad42da2e508611c33d7c4480b385", false},
	{"59066769b9ad42da2e508611c33d7c4480b3857k", false},
	{"5906676b99a4d2d2ae506811c33d7c4480b8357b", true},
}

func TestGenNeighbour(t *testing.T) {
	for _, test := range hashes {
		r := genNeighbour(test.s)
		if r != test.valid {
			t.Errorf("isValidInfoHash(%q) => %v expected %v", test.s, r, test.valid)
		}
	}
}

func TestIsValidInfoHash(t *testing.T) {
	for _, test := range hashes {
		r := isValidInfoHash(test.s)
		if r != test.valid {
			t.Errorf("isValidInfoHash(%q) => %v, expected %v", test.s, r, test.valid)
		}
	}
}

func TestDecodeInfoHash(t *testing.T) {
	for _, test := range hashes {
		_, err := decodeInfoHash(test.s)
		if (err == nil) != test.valid {
			t.Errorf("decodeInfoHash(%q) => %v expected %v", test.s, err, test.valid)
		}
	}
}
