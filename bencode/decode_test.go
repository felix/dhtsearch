package bencode

import (
	"testing"
)

func TestDecodeString(t *testing.T) {
	tests := []struct {
		in    string
		start int
		out   string
		n     int
	}{
		{in: "0:", start: 0, out: "", n: 2},
		{in: "5:hello", start: 0, out: "hello", n: 7},
		{in: "7:goodbye", start: 0, out: "goodbye", n: 9},
		{in: "11:hello world", start: 0, out: "hello world", n: 14},
		{in: "20:1-5%3~]+=\\| []>.,`??", start: 0, out: "1-5%3~]+=\\| []>.,`??", n: 23},
		{in: "123412347:goodbye", start: 8, out: "goodbye", n: 9},
	}

	for _, tt := range tests {
		r1, n, err := DecodeString([]byte(tt.in), tt.start)
		if err != nil {
			t.Errorf("DecodeString(%q) failed with error %q", tt.in, err)
		}

		if r1 != tt.out {
			t.Errorf("DecodeString(%q) => %q, expected %q", tt.in, r1, tt.out)
		}

		if n != tt.n {
			t.Errorf("DecodeString(%q) read %d, expected %d", tt.in, n, tt.n)
		}

		if tt.start == 0 {
			r2, err := Decode([]byte(tt.in))
			if err != nil {
				t.Errorf("Decode(%q) failed with error %q", tt.in, err)
			}

			if r2 != tt.out {
				t.Errorf("Decode(%q) => %q, expected %q", tt.in, r2, tt.out)
			}
		}

	}
}

func TestDecodeInt(t *testing.T) {
	tests := []struct {
		in    string
		start int
		out   int64
		n     int
	}{
		{in: "i0e", start: 0, out: int64(0), n: 3},
		{in: "i5e", start: 0, out: int64(5), n: 3},
		{in: "i-5e", start: 0, out: int64(-5), n: 4},
		{in: "i1234567890e", start: 0, out: int64(1234567890), n: 12},
		{in: "i-1234567890e", start: 0, out: int64(-1234567890), n: 13},
		{in: "asdfasdfi-5e", start: 8, out: int64(-5), n: 4},
	}

	for _, tt := range tests {
		r1, n, err := DecodeInt([]byte(tt.in), tt.start)
		if err != nil {
			t.Errorf("DecodeInt(%q) failed with error %q", tt.in, err)
		}

		if r1 != tt.out {
			t.Errorf("DecodeInt(%q) => %d, expected %d", tt.in, r1, tt.out)
		}

		if n != tt.n {
			t.Errorf("DecodeInt(%q) read %d, expected %d", tt.in, n, tt.n)
		}

		if tt.start == 0 {
			r2, err := Decode([]byte(tt.in))
			if err != nil {
				t.Errorf("Decode(%q) failed with error %q", tt.in, err)
			}

			if r2 != tt.out {
				t.Errorf("Decode(%q) => %d, expected %d", tt.in, r2, tt.out)
			}
		}

	}
}

func TestDecodeList(t *testing.T) {
	tests := []struct {
		in  string
		out []interface{}
		n   int
	}{
		{in: "l4:spam4:eggse", out: []interface{}{"spam", "eggs"}, n: 14},
		{in: "le", out: []interface{}{}, n: 2},
		{in: "li-1ei0ee", out: []interface{}{int64(-1), int64(0)}, n: 9},
		{in: "l4:testi-1ei0ee", out: []interface{}{"test", int64(-1), int64(0)}, n: 15},
	}

	for _, tt := range tests {
		r1, n, err := DecodeList([]byte(tt.in), 0)
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

		if n != tt.n {
			t.Errorf("DecodeList(%q) read %d, expected %d", tt.in, n, tt.n)
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
		in    string
		start int
		out   map[string]interface{}
		n     int
	}{
		{in: "d4:spam4:eggse", start: 0, out: map[string]interface{}{"spam": "eggs"}, n: 14},
		{in: "de", start: 0, out: map[string]interface{}{}, n: 2},
		{in: "d4:testi-1e3:twoi0ee", start: 0, out: map[string]interface{}{"test": int64(-1), "two": int64(0)}, n: 20},
		{in: "d4:testi0ee", start: 0, out: map[string]interface{}{"test": int64(0)}, n: 11},
		{in: "012345d4:spam4:eggse", start: 6, out: map[string]interface{}{"spam": "eggs"}, n: 14},
	}

	for _, tt := range tests {
		r1, _, err := DecodeDict([]byte(tt.in), tt.start)
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

		if tt.start == 0 {
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
