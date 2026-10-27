package manifest

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"shadowdevice/internal/protocol"
)

func fixture(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, protocol.Manifest) {
	t.Helper()
	pub, priv, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	m, e := Build(protocol.Manifest{Format: protocol.FormatVersion, Suite: protocol.Suite, DiskID: protocol.DiskID{1}, KeyEpoch: 1, Version: 1, DiskBytes: 1, BlockBytes: protocol.BlockSize, Entries: []protocol.ManifestEntry{{Generation: 1, BlockID: protocol.BlockID{1}}}})
	if e != nil {
		t.Fatal(e)
	}
	m, e = Sign(priv, m)
	if e != nil {
		t.Fatal(e)
	}
	return pub, priv, m
}
func TestAllAuthoritativeFieldsAuthenticated(t *testing.T) {
	pub, _, m := fixture(t)
	pin := Digest(m)
	if e := VerifyPinned(pub, m, pin); e != nil {
		t.Fatal(e)
	}
	if e := VerifyPinned(pub, m, pin); e != nil {
		t.Fatal("reopening same checkpoint must work", e)
	}
	changes := map[string]func(*protocol.Manifest){"disk": func(m *protocol.Manifest) { m.DiskID[0]++ }, "epoch": func(m *protocol.Manifest) { m.KeyEpoch++ }, "length": func(m *protocol.Manifest) { m.DiskBytes++ }, "parent": func(m *protocol.Manifest) { m.Parent[0]++ }, "version": func(m *protocol.Manifest) { m.Version++ }, "suite": func(m *protocol.Manifest) { m.Suite = "other" }, "format": func(m *protocol.Manifest) { m.Format++ }, "geometry": func(m *protocol.Manifest) { m.BlockBytes++ }, "generation": func(m *protocol.Manifest) { m.Entries[0].Generation++ }, "id": func(m *protocol.Manifest) { m.Entries[0].BlockID[0]++ }, "index": func(m *protocol.Manifest) { m.Entries[0].Index++ }, "root": func(m *protocol.Manifest) { m.MerkleRoot[0]++ }, "signature": func(m *protocol.Manifest) { m.Signature[0]++ }}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			bad := m
			bad.Entries = append([]protocol.ManifestEntry{}, m.Entries...)
			bad.Signature = append([]byte{}, m.Signature...)
			change(&bad)
			if Verify(pub, bad) == nil {
				t.Fatal("accepted tampering")
			}
		})
	}
	pub2, _, _ := ed25519.GenerateKey(rand.Reader)
	if Verify(pub2, m) == nil {
		t.Fatal("wrong signer")
	}
	wrong := pin
	wrong[0]++
	if VerifyPinned(pub, m, wrong) == nil {
		t.Fatal("wrong checkpoint")
	}
	if VerifyPinned(pub, m, protocol.Digest{}) == nil {
		t.Fatal("missing checkpoint")
	}
}
func TestGeometryAndEmpty(t *testing.T) {
	_, priv, m := fixture(t)
	m.DiskBytes = 0
	m.Entries = nil
	empty, e := Build(m)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Sign(priv, empty); e != nil {
		t.Fatal(e)
	}
	m.DiskBytes = protocol.BlockSize + 1
	m.Entries = []protocol.ManifestEntry{{Index: 0, Generation: 1, BlockID: protocol.BlockID{1}}, {Index: 0, Generation: 1, BlockID: protocol.BlockID{2}}}
	if _, e = Build(m); e == nil {
		t.Fatal("duplicate index")
	}
	m.Entries = nil
	m.DiskBytes = uint64(protocol.MaxBlocks)*protocol.BlockSize + 1
	if _, e = Build(m); e == nil {
		t.Fatal("oversized geometry")
	}
}
