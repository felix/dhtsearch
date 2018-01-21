package bencode

import (
	"bytes"
	"testing"

	alt1 "github.com/marksamman/bencode"
)

func BenchmarkMyStringDecode(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Decode([]byte("11:hello world"))
	}
}

func BenchmarkMyIntDecode(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Decode([]byte("i1234234e"))
	}
}

func BenchmarkMyListDecode(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Decode([]byte("l4:spam4:eggse"))
	}
}

func BenchmarkMyDictDecode(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Decode([]byte("d4:spam4:eggse"))
	}
}

func BenchmarkAlt1StringDecode(b *testing.B) {
	for n := 0; n < b.N; n++ {
		alt1.Decode(bytes.NewBufferString("11:hello world"))
	}
}

func BenchmarkAlt1IntDecode(b *testing.B) {
	for n := 0; n < b.N; n++ {
		alt1.Decode(bytes.NewBufferString("i1234234e"))
	}
}

func BenchmarkAlt1ListDecode(b *testing.B) {
	for n := 0; n < b.N; n++ {
		alt1.Decode(bytes.NewBufferString("l4:spam4:eggse"))
	}
}

func BenchmarkAlt1DictDecode(b *testing.B) {
	for n := 0; n < b.N; n++ {
		alt1.Decode(bytes.NewBufferString("d4:spam4:eggse"))
	}
}
