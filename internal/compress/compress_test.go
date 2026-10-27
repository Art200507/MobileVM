package compress

import (
	"bytes"
	"compress/flate"
	"crypto/rand"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	cases := map[string][]byte{
		"empty":         {},
		"small":         []byte("shadowdevice"),
		"all zero 4MiB": make([]byte, 4<<20),
	}
	random := make([]byte, 4<<20)
	if _, e := rand.Read(random); e != nil {
		t.Fatal(e)
	}
	cases["random 4MiB"] = random
	for name, p := range cases {
		t.Run(name, func(t *testing.T) {
			c, e := Compress(p, flate.BestCompression)
			if e != nil {
				t.Fatal(e)
			}
			got, e := Decompress(c, len(p))
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(got, p) {
				t.Fatal("round trip mismatch")
			}
		})
	}
}

func TestZeroBlockCompressesSmall(t *testing.T) {
	p := make([]byte, 4<<20)
	c, e := Compress(p, flate.BestCompression)
	if e != nil {
		t.Fatal(e)
	}
	if len(c) > 4096 {
		t.Fatalf("expected a sparse 4MiB block to compress well under 4KiB, got %d bytes", len(c))
	}
}

// TestPaddedTailIsIgnored proves the property docs/product-expansion.md
// relies on: a compressed buffer padded out to a fixed size still decodes
// to exactly the original plaintext, because Decompress stops at the
// stream's own end marker rather than reading the whole buffer.
func TestPaddedTailIsIgnored(t *testing.T) {
	p := []byte("fixed-size tier padding should be invisible to the reader")
	c, e := Compress(p, flate.BestCompression)
	if e != nil {
		t.Fatal(e)
	}
	const tier = 1 << 20
	padded := make([]byte, tier)
	copy(padded, c)
	got, e := Decompress(padded, len(p))
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(got, p) {
		t.Fatal("padding was not ignored", got)
	}
}

func TestDecompressRejectsOversizedOutput(t *testing.T) {
	p := bytes.Repeat([]byte{7}, 1<<20)
	c, e := Compress(p, flate.BestCompression)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Decompress(c, len(p)-1); e != ErrTooLarge {
		t.Fatal("expected ErrTooLarge", e)
	}
	// The exact bound must still succeed.
	if _, e = Decompress(c, len(p)); e != nil {
		t.Fatal(e)
	}
}

func FuzzDecompress(f *testing.F) {
	c, _ := Compress([]byte("seed corpus entry"), flate.BestCompression)
	f.Add(c, 4<<20)
	f.Add([]byte{}, 4<<20)
	f.Add([]byte{0xff, 0x00, 0xff}, 16)
	f.Fuzz(func(t *testing.T, p []byte, maxOutput int) {
		if maxOutput < 0 || maxOutput > 16<<20 {
			return
		}
		out, e := Decompress(p, maxOutput)
		if e != nil {
			return
		}
		if len(out) > maxOutput {
			t.Fatal("exceeded requested bound without error")
		}
	})
}
