package bencode

import (
	"testing"
)

func TestDecodeString(t *testing.T) {
	tests := []struct {
		in  string
		out string
		end int
	}{
		{in: "0:", out: "", end: 1},
		{in: "5:hello", out: "hello", end: 6},
		{in: "7:goodbye", out: "goodbye", end: 8},
		{in: "11:hello world", out: "hello world", end: 13},
		{in: "20:1-5%3~]+=\\| []>.,`??", out: "1-5%3~]+=\\| []>.,`??", end: 22},
	}

	for _, tt := range tests {
		r1, end, err := DecodeString([]byte(tt.in), 0)
		if err != nil {
			t.Errorf("DecodeString(%q) failed with error %q", tt.in, err)
		}

		if r1 != tt.out {
			t.Errorf("DecodeString(%q) => %q, expected %q", tt.in, r1, tt.out)
		}

		if end != tt.end {
			t.Errorf("DecodeString(%q) ended at %d, expected %d", tt.in, end, tt.end)
		}

		r2, err := Decode([]byte(tt.in))
		if err != nil {
			t.Errorf("DecodeString(%q) failed with error %q", tt.in, err)
		}

		if r2 != tt.out {
			t.Errorf("DecodeString(%q) => %q, expected %q", tt.in, r2, tt.out)
		}

	}
}

func TestDecodeInt(t *testing.T) {
	tests := []struct {
		in  string
		out int64
		end int
	}{
		{in: "i0e", out: int64(0), end: 2},
		{in: "i5e", out: int64(5), end: 2},
		{in: "i-5e", out: int64(-5), end: 3},
		{in: "i1234567890e", out: int64(1234567890), end: 11},
		{in: "i-1234567890e", out: int64(-1234567890), end: 12},
	}

	for _, tt := range tests {
		r1, end, err := DecodeInt([]byte(tt.in), 0)
		if err != nil {
			t.Errorf("DecodeInt(%q) failed with error %q", tt.in, err)
		}

		if r1 != tt.out {
			t.Errorf("DecodeInt(%q) => %d, expected %d", tt.in, r1, tt.out)
		}

		if end != tt.end {
			t.Errorf("DecodeInt(%q) ended at %d, expected %d", tt.in, end, tt.end)
		}

		r2, err := Decode([]byte(tt.in))
		if err != nil {
			t.Errorf("DecodeInt(%q) failed with error %q", tt.in, err)
		}

		if r2 != tt.out {
			t.Errorf("DecodeInt(%q) => %d, expected %d", tt.in, r2, tt.out)
		}

	}
}

func TestDecodeList(t *testing.T) {
	tests := []struct {
		in  string
		out []interface{}
		end int
	}{
		{in: "l4:spam4:eggse", out: []interface{}{"spam", "eggs"}, end: 13},
		{in: "le", out: []interface{}{}, end: 1},
		{in: "li-1ei0ee", out: []interface{}{int64(-1), int64(0)}, end: 8},
		{in: "l4:testi-1ei0ee", out: []interface{}{"test", int64(-1), int64(0)}, end: 14},
	}

	for _, tt := range tests {
		r1, end, err := DecodeList([]byte(tt.in), 0)
		if err != nil {
			t.Errorf("DecodeList(%q) failed with error %q", tt.in, err)
		}

		if len(r1) != len(tt.out) {
			t.Errorf("DecodeList(%q) => %d items, expected %d", tt.in, len(r1), len(tt.out))
		}

		for i := range r1 {
			if r1[i] != tt.out[i] {
				t.Errorf("DecodeList(%q) => %v, expected %v", tt.in, r1, tt.out)
			}
		}

		if end != tt.end {
			t.Errorf("DecodeList(%q) ended at %d, expected %d", tt.in, end, tt.end)
		}

		r2, err := Decode([]byte(tt.in))
		if err != nil {
			t.Errorf("Decode(%q) failed with error %q", tt.in, err)
		}

		r3, ok := r2.([]interface{})
		if !ok {
			t.Errorf("Decode(%q) did not return a slice", tt.in)
		}

		for i := range r3 {
			if r3[i] != tt.out[i] {
				t.Errorf("Decode(%q) => %v, expected %v", tt.in, r3, tt.out)
			}
		}
	}
}

func TestDecodeDict(t *testing.T) {
	tests := []struct {
		in  string
		out map[string]interface{}
		end int
	}{
		{in: "d4:spam4:eggse", out: map[string]interface{}{"spam": "eggs"}, end: 13},
		{in: "de", out: map[string]interface{}{}, end: 1},
		{in: "d4:testi-1e3:twoi0ee", out: map[string]interface{}{"test": int64(-1), "two": int64(0)}, end: 14},
		{in: "d4:testi0ee", out: map[string]interface{}{"test": int64(0)}, end: 10},
	}

	for _, tt := range tests {
		r1, _, err := DecodeDict([]byte(tt.in), 0)
		if err != nil {
			t.Errorf("DecodeDict(%q) failed with error %q", tt.in, err)
		}

		if len(r1) != len(tt.out) {
			t.Errorf("DecodeDict(%q) => %d items, expected %d", tt.in, len(r1), len(tt.out))
		}

		for i := range r1 {
			if r1[i] != tt.out[i] {
				t.Errorf("DecodeDict(%q) => %v, expected %v", tt.in, r1, tt.out)
			}
		}

		r2, err := Decode([]byte(tt.in))
		if err != nil {
			t.Errorf("Decode(%q) failed with error %q", tt.in, err)
		}

		r3, ok := r2.(map[string]interface{})
		if !ok {
			t.Errorf("Decode(%q) did not return a map", tt.in)
		}

		for k := range r3 {
			if r3[k] != tt.out[k] {
				t.Errorf("Decode(%q) => %v, expected %v", tt.in, r3, tt.out)
			}
		}
	}
}

func TestReadUntil(t *testing.T) {
	tests := []struct {
		in    string
		start int
		out   string
		end   int
	}{
		{in: "0:", start: 0, out: "0", end: 2},
		{in: "5:hello", start: 0, out: "5", end: 2},
		{in: "1234567:goodbye", start: 0, out: "1234567", end: 8},
		{in: "asdfasdfsa5:hello", start: 10, out: "5", end: 12},
	}

	for _, tt := range tests {
		r, i, err := readUntil([]byte(tt.in), tt.start, ':')
		if err != nil {
			t.Errorf("readUntil(%q) failed with error %q", tt.in, err)
		}

		if string(r) != tt.out {
			t.Errorf("readUntil(%q) => %q, expected %q", tt.in, r, tt.out)
		}

		if i != tt.end {
			t.Errorf("readUntil(%q) ended at %d, expected %d", tt.in, i, tt.end)
		}
	}
}

func BenchmarkDecodeWithString(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Decode([]byte("11:hello world"))
	}
}

func BenchmarkDecodeString(b *testing.B) {
	for n := 0; n < b.N; n++ {
		DecodeString([]byte("11:hello world"), 0)
	}
}

func BenchmarkDecodeWithInt(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Decode([]byte("i1234234e"))
	}
}

func BenchmarkDecodeInt(b *testing.B) {
	for n := 0; n < b.N; n++ {
		DecodeInt([]byte("i1234234e"), 0)
	}
}

func BenchmarkDecodeWithList(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Decode([]byte("l4:spam4:eggse"))
	}
}

func BenchmarkDecodeList(b *testing.B) {
	for n := 0; n < b.N; n++ {
		DecodeList([]byte("l4:spam4:eggse"), 0)
	}
}

func BenchmarkDecodeWithDict(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Decode([]byte("d4:spam4:eggse"))
	}
}

func BenchmarkDecodeDict(b *testing.B) {
	for n := 0; n < b.N; n++ {
		DecodeDict([]byte("d4:spam4:eggse"), 0)
	}
}
