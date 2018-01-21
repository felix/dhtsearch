package bencode

import (
	"testing"
)

func TestEncodeString(t *testing.T) {
	tests := []struct {
		in  string
		out string
		end int
	}{
		{in: "", out: "0:"},
		{in: "hello", out: "5:hello"},
		{in: "goodbye", out: "7:goodbye"},
		{in: "hello world", out: "11:hello world"},
		{in: "1-5%3~]+=\\| []>.,`??", out: "20:1-5%3~]+=\\| []>.,`??"},
	}

	for _, tt := range tests {
		r1b, err := EncodeString(tt.in)
		if err != nil {
			t.Errorf("EncodeString(%q) failed with error %q", tt.in, err)
		}

		r1 := string(r1b)

		if r1 != tt.out {
			t.Errorf("EncodeString(%q) => %q, expected %q", tt.in, r1, tt.out)
		}

		r2b, err := Encode(tt.in)
		if err != nil {
			t.Errorf("EncodeString(%q) failed with error %q", tt.in, err)
		}

		r2 := string(r2b)
		if r2 != tt.out {
			t.Errorf("EncodeString(%q) => %q, expected %q", tt.in, r2, tt.out)
		}
	}
}

func TestEncodeInt(t *testing.T) {
	tests := []struct {
		in  interface{}
		out string
	}{
		{out: "i0e", in: int64(0)},
		{out: "i0e", in: int32(0)},
		{out: "i0e", in: int16(0)},
		{out: "i0e", in: 0},
		{out: "i5e", in: int64(5)},
		{out: "i5e", in: int32(5)},
		{out: "i5e", in: 5},
		{out: "i-5e", in: int64(-5)},
		{out: "i-5e", in: int32(-5)},
		{out: "i-5e", in: -5},
		{out: "i1234567890e", in: int64(1234567890)},
		{out: "i-1234567890e", in: int64(-1234567890)},
	}

	var r1b []byte
	var err error

	for _, tt := range tests {
		switch v := tt.in.(type) {
		case int:
			r1b, err = EncodeInt(int64(v))
		case int16:
			r1b, err = EncodeInt(int64(v))
		case int32:
			r1b, err = EncodeInt(int64(v))
		case int64:
			r1b, err = EncodeInt(v)
		}
		if err != nil {
			t.Errorf("EncodeInt(%d) failed with error %q", tt.in, err)
		}
		r1 := string(r1b)

		if r1 != tt.out {
			t.Errorf("EncodeInt(%d) => %s, expected %s", tt.in, r1, tt.out)
		}

		r2b, err := Encode(tt.in)
		if err != nil {
			t.Errorf("EncodeInt(%d) failed with error %q", tt.in, err)
		}
		r2 := string(r2b)

		if r2 != tt.out {
			t.Errorf("EncodeInt(%d) => %s, expected %s", tt.in, r2, tt.out)
		}
	}
}

func TestEncodeList(t *testing.T) {
	tests := []struct {
		in  []interface{}
		out string
	}{
		{out: "l4:spam4:eggse", in: []interface{}{"spam", "eggs"}},
		{out: "le", in: []interface{}{}},
		{out: "li-1ei0ee", in: []interface{}{int64(-1), int64(0)}},
		{out: "l4:testi-1ei0ee", in: []interface{}{"test", int64(-1), int64(0)}},
	}

	for _, tt := range tests {
		r1b, err := EncodeList(tt.in)
		if err != nil {
			t.Errorf("EncodeList(%v) failed with error %q", tt.in, err)
		}
		r1 := string(r1b)

		if r1 != tt.out {
			t.Errorf("EncodeList(%v) => %s, expected %s", tt.in, r1, tt.out)
		}

		r2b, err := Encode(tt.in)
		if err != nil {
			t.Errorf("Encode(%v) failed with error %q", tt.in, err)
		}
		r2 := string(r2b)

		if r2 != tt.out {
			t.Errorf("Encode(%v) => %s, expected %s", tt.in, r2, tt.out)
		}
	}
}

func TestEncodeDict(t *testing.T) {
	tests := []struct {
		in  map[string]interface{}
		out string
	}{
		{out: "d4:spam4:eggse", in: map[string]interface{}{"spam": "eggs"}},
		{out: "de", in: map[string]interface{}{}},
		{out: "d4:testi-1e3:twoi0ee", in: map[string]interface{}{"test": int64(-1), "two": int64(0)}},
		{out: "d4:testi0ee", in: map[string]interface{}{"test": int64(0)}},
	}

	for _, tt := range tests {
		r1b, err := EncodeDict(tt.in)
		if err != nil {
			t.Errorf("EncodeDict(%v) failed with error %q", tt.in, err)
		}
		r1 := string(r1b)

		if r1 != tt.out {
			t.Errorf("EncodeDict(%v) => %s, expected %s", tt.in, r1, tt.out)
		}

		r2b, err := Encode(tt.in)
		if err != nil {
			t.Errorf("Encode(%v) failed with error %q", tt.in, err)
		}
		r2 := string(r2b)

		if r2 != tt.out {
			t.Errorf("Encode(%v) => %s, expected %s", tt.in, r2, tt.out)
		}
	}
}
