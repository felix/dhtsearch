package dht

import (
	"encoding/hex"
	"testing"
)

func TestInfohashImport(t *testing.T) {
	var ih Infohash

	idHex := "5a3ce1c14e7a08645677bbd1cfe7d8f956d53256"
	err := ih.FromString(idHex)
	if err != nil {
		t.Errorf("FromString failed with %s", err)
	}

	idBytes, err := hex.DecodeString(idHex)

	ih2 := Infohash(idBytes)
	if !ih.Equal(ih2) {
		t.Errorf("expected %s to equal %s", ih, ih2)
	}
}

func TestInfohashLength(t *testing.T) {
	ih := randomInfoHash()
	if len(ih) != 20 {
		t.Errorf("%s as string should be length 20, got %d", ih, len(ih))
	}
}
