package store

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"

	"shadowdevice/internal/manifest"
	"shadowdevice/internal/protocol"
)

// FuzzDecodeManifest exercises the untrusted JSON boundary a manifest crosses
// before any signature check runs. It must never panic or exhaust memory, and
// any input that both decodes and verifies against the original signer must
// be byte-identical to what was actually signed: a mutated field can only
// pass Verify by accident if the fuzzer rediscovers the exact original bytes.
func FuzzDecodeManifest(f *testing.F) {
	pub, priv, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		f.Fatal(e)
	}
	m, e := manifest.Build(protocol.Manifest{Format: protocol.FormatVersion, Suite: protocol.Suite, DiskID: protocol.DiskID{1}, KeyEpoch: 1, Version: 1, DiskBytes: 1, BlockBytes: protocol.BlockSize, Entries: []protocol.ManifestEntry{{Generation: 1, BlockID: protocol.BlockID{1}}}})
	if e != nil {
		f.Fatal(e)
	}
	m, e = manifest.Sign(priv, m)
	if e != nil {
		f.Fatal(e)
	}
	seed, e := json.Marshal(m)
	if e != nil {
		f.Fatal(e)
	}
	want := manifest.Digest(m)
	f.Add(seed)
	f.Add([]byte(""))
	f.Add([]byte("{}"))
	f.Add([]byte("null"))
	f.Add([]byte(`{"Entries":[{},{},{},{},{},{},{},{},{},{}]}`))
	f.Add([]byte(`{"DiskBytes":18446744073709551615,"BlockBytes":4294967295}`))
	f.Fuzz(func(t *testing.T, p []byte) {
		decoded, e := DecodeManifest(p)
		if e != nil {
			return
		}
		if e = manifest.Verify(pub, decoded); e != nil {
			return
		}
		if manifest.Digest(decoded) != want {
			t.Fatal("verified manifest diverged from the only bytes ever signed")
		}
	})
}
