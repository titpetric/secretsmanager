package ulid

import (
	"strings"
	"testing"
	"time"
)

// TestEncode pins the encoding against vectors taken from oklog/ulid, which
// this package replaced. The whole point of a ULID is that everyone writes
// the same 128 bits the same way.
func TestEncode(t *testing.T) {
	for _, test := range []struct {
		raw  [16]byte
		want string
	}{
		{
			raw:  [16]byte{},
			want: "00000000000000000000000000",
		},
		{
			raw: [16]byte{
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
			want: "7ZZZZZZZZZZZZZZZZZZZZZZZZZ",
		},
		{
			raw: [16]byte{
				0xd0, 0x57, 0x38, 0xa6, 0x1c, 0x30, 0x34, 0x37,
				0xc9, 0x4b, 0x64, 0x48, 0xa7, 0xbf, 0xb2, 0xc1,
			},
			want: "6GAWWAC71G6GVWJJV492KVZCP1",
		},
		{
			raw: [16]byte{
				0xab, 0x03, 0x4e, 0xc5, 0xe0, 0x18, 0x16, 0xfe,
				0x9b, 0x8f, 0xde, 0xa2, 0xa1, 0x58, 0x4f, 0x06,
			},
			want: "5B0D7CBR0R2VZ9Q3YYMAGNGKR6",
		},
		{
			raw: [16]byte{
				0x0e, 0x3b, 0x24, 0x8a, 0x17, 0x89, 0x68, 0xbc,
				0x28, 0x8d, 0x0d, 0x6a, 0xc0, 0xfe, 0xdc, 0x2f,
			},
			want: "0E7CJ8M5W9D2Y2H38DDB0FXQ1F",
		},
	} {
		if got := encode(test.raw); got != test.want {
			t.Errorf("encode(%x) = %q, want %q", test.raw, got, test.want)
		}
	}
}

// TestNew covers the ULIDs this package makes.
func TestNew(t *testing.T) {
	when := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)

	first, second := New(when), New(when)
	if first == second {
		t.Error("two ULIDs for one time came out the same, so the entropy isn't random")
	}

	// The timestamp is in front, so ULIDs made at one time share a prefix
	// and ULIDs made later sort after them.
	if got, want := first[:10], second[:10]; got != want {
		t.Errorf("timestamps differ: %q and %q", got, want)
	}
	if later := New(when.Add(time.Second)); later <= first {
		t.Errorf("%q was made later than %q but doesn't sort after it", later, first)
	}

	for _, id := range []string{first, second, Make()} {
		if len(id) != Length {
			t.Errorf("%q is %d characters, want %d", id, len(id), Length)
		}
		if !Valid(id) {
			t.Errorf("Valid(%q) = false", id)
		}
		if strings.ContainsAny(id, "ILOU") {
			t.Errorf("%q holds a character Crockford base32 leaves out", id)
		}
	}
}

// TestMake covers a ULID for the time it was asked for.
func TestMake(t *testing.T) {
	earlier := New(time.Now().Add(-time.Second))

	id := Make()
	if !Valid(id) {
		t.Fatalf("Make() = %q, which isn't a ULID", id)
	}
	if id <= earlier {
		t.Errorf("Make() = %q, want it to sort after %q, made a second earlier", id, earlier)
	}
}

// TestValid covers what passes for a ULID.
func TestValid(t *testing.T) {
	for id, want := range map[string]bool{
		"0E7CJ8M5W9D2Y2H38DDB0FXQ1F":  true,
		"00000000000000000000000000":  true,
		"7ZZZZZZZZZZZZZZZZZZZZZZZZZ":  true,
		"":                            false,
		"0E7CJ8M5W9D2Y2H38DDB0FXQ1":   false, // one short
		"0E7CJ8M5W9D2Y2H38DDB0FXQ1FF": false, // one long
		"0E7CJ8M5W9D2Y2H38DDB0FXQ1i":  false, // not in the alphabet
		"8E7CJ8M5W9D2Y2H38DDB0FXQ1F":  false, // more than 128 bits
		"25349927-99b2-4ac5-ad59-d63": false, // a UUID, which files still hold
	} {
		if got := Valid(id); got != want {
			t.Errorf("Valid(%q) = %v, want %v", id, got, want)
		}
	}
}
