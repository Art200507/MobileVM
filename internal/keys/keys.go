// Package keys derives disk-scoped development keys from a portable data root.
// Device wrapping and protected memory are separate, unimplemented layers.
package keys

import (
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"

	"shadowdevice/internal/protocol"
)

const MasterKeySize = 32

type MasterKey [MasterKeySize]byte
type BlockKey [32]byte
type BlockIDKey [32]byte
type SessionCacheKey [32]byte

func GenerateMasterKey() (MasterKey, error) { var k MasterKey; _, e := rand.Read(k[:]); return k, e }
func derive(mk MasterKey, disk protocol.DiskID, epoch uint64, label string) ([32]byte, error) {
	salt := append([]byte{}, disk[:]...)
	salt = binary.BigEndian.AppendUint64(salt, epoch)
	raw, e := hkdf.Key(sha256.New, mk[:], salt, "shadowdevice/v1/"+label, 32)
	var out [32]byte
	copy(out[:], raw)
	clear(raw)
	return out, e
}
func DeriveBlockKey(mk MasterKey, disk protocol.DiskID, epoch uint64) (BlockKey, error) {
	k, e := derive(mk, disk, epoch, "blocks")
	return BlockKey(k), e
}
func DeriveBlockIDKey(mk MasterKey, disk protocol.DiskID, epoch uint64) (BlockIDKey, error) {
	k, e := derive(mk, disk, epoch, "ids")
	return BlockIDKey(k), e
}
func NewSessionCacheKey() (SessionCacheKey, error) {
	var k SessionCacheKey
	_, e := rand.Read(k[:])
	return k, e
}

// Shred clears this value only. Copies, runtime buffers and forensic erasure are
// not covered; this is not an enforced session revocation mechanism.
func (k *SessionCacheKey) Shred() { clear(k[:]) }
