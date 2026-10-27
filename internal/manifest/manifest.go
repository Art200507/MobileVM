// Package manifest authenticates disk geometry, history and immutable entries.
package manifest

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"shadowdevice/internal/protocol"
)

var ErrInvalid = errors.New("manifest: invalid format, geometry, signature or checkpoint")

func entryBytes(e protocol.ManifestEntry) []byte {
	p := binary.BigEndian.AppendUint64(nil, e.Index)
	p = binary.BigEndian.AppendUint64(p, e.Generation)
	return append(p, e.BlockID[:]...)
}
func root(es []protocol.ManifestEntry) [32]byte {
	if len(es) == 0 {
		return sha256.Sum256([]byte("shadowdevice/merkle/empty/v1"))
	}
	nodes := make([][32]byte, len(es))
	for i, e := range es {
		nodes[i] = sha256.Sum256(append([]byte{0}, entryBytes(e)...))
	}
	for len(nodes) > 1 {
		next := make([][32]byte, 0, (len(nodes)+1)/2)
		for i := 0; i < len(nodes); i += 2 {
			right := nodes[i]
			if i+1 < len(nodes) {
				right = nodes[i+1]
			}
			p := append([]byte{1}, nodes[i][:]...)
			p = append(p, right[:]...)
			next = append(next, sha256.Sum256(p))
		}
		nodes = next
	}
	return nodes[0]
}
func validate(m protocol.Manifest) error {
	if m.Format != protocol.FormatVersion || m.Suite != protocol.Suite || m.DiskID == (protocol.DiskID{}) || m.KeyEpoch == 0 || m.Version == 0 || m.BlockBytes != protocol.BlockSize || m.DiskBytes > uint64(protocol.MaxBlocks)*protocol.BlockSize || len(m.Entries) > protocol.MaxBlocks {
		return ErrInvalid
	}
	n := m.DiskBytes / protocol.BlockSize
	if m.DiskBytes%protocol.BlockSize != 0 {
		n++
	}
	if uint64(len(m.Entries)) != n {
		return ErrInvalid
	}
	if (m.Version == 1) != (m.Parent == (protocol.Digest{})) {
		return ErrInvalid
	}
	for i, e := range m.Entries {
		if e.Index != uint64(i) || e.Generation == 0 || e.Generation > m.Version || e.BlockID == (protocol.BlockID{}) {
			return ErrInvalid
		}
	}
	if m.MerkleRoot != root(m.Entries) {
		return ErrInvalid
	}
	return nil
}

// Build copies entries, computes their ordered Merkle root and validates geometry.
func Build(m protocol.Manifest) (protocol.Manifest, error) {
	m.Entries = append([]protocol.ManifestEntry{}, m.Entries...)
	m.Signature = nil
	m.MerkleRoot = root(m.Entries)
	return m, validate(m)
}
func unsigned(m protocol.Manifest) []byte {
	p := []byte("shadowdevice/manifest/v1\x00")
	p = binary.BigEndian.AppendUint32(p, m.Format)
	p = binary.BigEndian.AppendUint32(p, uint32(len(m.Suite)))
	p = append(p, m.Suite...)
	p = append(p, m.DiskID[:]...)
	for _, v := range []uint64{m.KeyEpoch, m.Version} {
		p = binary.BigEndian.AppendUint64(p, v)
	}
	p = append(p, m.Parent[:]...)
	p = binary.BigEndian.AppendUint64(p, m.DiskBytes)
	p = binary.BigEndian.AppendUint32(p, m.BlockBytes)
	p = binary.BigEndian.AppendUint64(p, uint64(len(m.Entries)))
	return append(p, m.MerkleRoot[:]...)
}
func Sign(priv ed25519.PrivateKey, m protocol.Manifest) (protocol.Manifest, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return m, ErrInvalid
	}
	if e := validate(m); e != nil {
		return m, e
	}
	m.Signature = ed25519.Sign(priv, unsigned(m))
	return m, nil
}
func Verify(pub ed25519.PublicKey, m protocol.Manifest) error {
	if e := validate(m); e != nil {
		return e
	}
	if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, unsigned(m), m.Signature) {
		return ErrInvalid
	}
	return nil
}

// Digest is a canonical hash of signed metadata, independent of transport JSON.
func Digest(m protocol.Manifest) protocol.Digest {
	return protocol.Digest(sha256.Sum256(append(unsigned(m), m.Signature...)))
}

// VerifyPinned accepts only the exact independently supplied checkpoint. It does
// not discover the latest snapshot or trust a key supplied by the block store.
func VerifyPinned(pub ed25519.PublicKey, m protocol.Manifest, want protocol.Digest) error {
	if e := Verify(pub, m); e != nil {
		return e
	}
	if want == (protocol.Digest{}) || Digest(m) != want {
		return ErrInvalid
	}
	return nil
}
