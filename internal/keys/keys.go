// Package keys implements the key hierarchy described in docs/architecture.md
// (Master key -> block key / block-ID key / signing key via HKDF, plus
// per-session cache keys). Implementation belongs to Phase 1 (block keys)
// and Phase 3 (session cache keys); this file only fixes the shape.
package keys

import "errors"

var errNotImplemented = errors.New("keys: not implemented yet (see docs/architecture.md Build plan)")

// MasterKeySize is the size in bytes of the random 256-bit master key.
const MasterKeySize = 32

// MasterKey is the single random key that everything else derives from.
// It is generated on the owner's phone and never seen by the server.
type MasterKey [MasterKeySize]byte

// BlockKey encrypts the contents of every disk block. Derived as
// HKDF(MK, "blocks").
type BlockKey [32]byte

// BlockIDKey derives keyed block identifiers so the cloud can't spot
// identical blocks across snapshots or accounts. Derived as
// HKDF(MK, "ids").
type BlockIDKey [32]byte

// SessionCacheKey protects blocks cached in the protected VM's memory on a
// borrowed phone. Random per session; destroyed (crypto-shredded) when the
// session ends. See Phase 3.
type SessionCacheKey [32]byte

// GenerateMasterKey produces a new random 256-bit master key on the owner's
// device. Phase 1.
func GenerateMasterKey() (MasterKey, error) {
	return MasterKey{}, errNotImplemented
}

// DeriveBlockKey derives the block content key from the master key via
// HKDF-SHA256 with info "blocks". Phase 1.
func DeriveBlockKey(mk MasterKey) (BlockKey, error) {
	return BlockKey{}, errNotImplemented
}

// DeriveBlockIDKey derives the block-ID key from the master key via
// HKDF-SHA256 with info "ids". Phase 1.
func DeriveBlockIDKey(mk MasterKey) (BlockIDKey, error) {
	return BlockIDKey{}, errNotImplemented
}

// NewSessionCacheKey generates a fresh random session cache key inside the
// protected VM. Phase 3.
func NewSessionCacheKey() (SessionCacheKey, error) {
	return SessionCacheKey{}, errNotImplemented
}

// Shred overwrites and discards a session cache key when a session ends,
// so cached blocks on the borrowed phone become unreadable. Phase 3.
func (k *SessionCacheKey) Shred() {
	// TODO(phase 3): zero k in place; caller drops the reference.
}
