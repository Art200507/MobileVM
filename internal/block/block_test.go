package block

import (
	"bytes"
	"testing"

	"shadowdevice/internal/keys"
	"shadowdevice/internal/protocol"
)

func TestSealExpectedContextAndTampering(t *testing.T) {
	var mk keys.MasterKey
	mk[0] = 1
	disk := protocol.DiskID{1}
	bk, _ := keys.DeriveBlockKey(mk, disk, 1)
	ik, _ := keys.DeriveBlockIDKey(mk, disk, 1)
	c := protocol.Context{DiskID: disk, KeyEpoch: 1, Generation: 1}
	plain := bytes.Repeat([]byte{42}, protocol.BlockSize)
	b, id, e := Seal(bk, ik, c, plain)
	if e != nil {
		t.Fatal(e)
	}
	encoded, e := Marshal(b)
	if e != nil {
		t.Fatal(e)
	}
	if len(encoded) != ObjectSize {
		t.Fatal(len(encoded))
	}
	decoded, e := Unmarshal(encoded)
	if e != nil {
		t.Fatal(e)
	}
	got, e := Open(bk, ik, c, id, decoded)
	if e != nil || !bytes.Equal(got, plain) {
		t.Fatal("roundtrip", e)
	}
	second, id2, e := Seal(bk, ik, c, plain)
	if e != nil || id == id2 || b.Salt == second.Salt {
		t.Fatal("seal must randomize", e)
	}
	for _, which := range []string{"disk", "epoch", "index", "generation", "salt", "ciphertext", "nonce", "id", "key"} {
		t.Run(which, func(t *testing.T) {
			bad := decoded
			bad.Sealed = bytes.Clone(decoded.Sealed)
			expect := c
			want := id
			key := bk
			switch which {
			case "disk":
				expect.DiskID[0]++
			case "epoch":
				expect.KeyEpoch++
			case "index":
				expect.Index++
			case "generation":
				expect.Generation++
			case "salt":
				bad.Salt[0] ^= 1
			case "ciphertext":
				bad.Sealed[40] ^= 1
			case "nonce":
				bad.Sealed[0] ^= 1
			case "id":
				want[0] ^= 1
			case "key":
				key[0] ^= 1
			}
			out, e := Open(key, ik, expect, want, bad)
			if e == nil || out != nil {
				t.Fatal("accepted tampering")
			}
		})
	}
	if _, e = Unmarshal(encoded[:len(encoded)-1]); e == nil {
		t.Fatal("accepted truncated object")
	}
}
func TestKeysAreScoped(t *testing.T) {
	var mk keys.MasterKey
	a, _ := keys.DeriveBlockKey(mk, protocol.DiskID{1}, 1)
	b, _ := keys.DeriveBlockKey(mk, protocol.DiskID{2}, 1)
	c, _ := keys.DeriveBlockKey(mk, protocol.DiskID{1}, 2)
	d, _ := keys.DeriveBlockIDKey(mk, protocol.DiskID{1}, 1)
	if a == b || a == c || [32]byte(a) == [32]byte(d) {
		t.Fatal("keys lack domain separation")
	}
}

func FuzzMalformedObject(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{1, 2, 3, 4})
	f.Fuzz(func(t *testing.T, p []byte) {
		b, e := Unmarshal(p)
		if e != nil {
			return
		}
		encoded, e := Marshal(b)
		if e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(encoded, p) {
			t.Fatal("non-canonical roundtrip")
		}
	})
}
