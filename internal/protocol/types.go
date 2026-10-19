// Package protocol defines the wire and domain types shared between the
// sync agent, block store, guest client, and manifest packages. See
// docs/architecture.md for the design these types implement.
package protocol

import "time"

// BlockSize is the fixed size of every encrypted disk block, chosen so that
// block count and sizes don't leak which files are present (see "Blocks" in
// docs/architecture.md).
const BlockSize = 4 << 20 // 4 MB

// BlockID is a keyed (HMAC) identifier for a block, derived from the
// block-ID key so the cloud cannot correlate identical blocks across users
// or snapshots.
type BlockID [32]byte

// EncryptedBlock is a single sealed block as stored by the block store.
// Nonce and ciphertext are opaque to the server; AAD binds Index and
// SnapshotVersion so the server cannot swap blocks or replay an old one.
type EncryptedBlock struct {
	ID              BlockID
	Index           uint64
	SnapshotVersion uint64
	Nonce           []byte
	Ciphertext      []byte // includes the AEAD auth tag
}

// ManifestEntry maps a logical block index to the current BlockID holding
// its contents, for a given snapshot version.
type ManifestEntry struct {
	Index   uint64
	BlockID BlockID
}

// Manifest is the signed, monotonically versioned list of block entries for
// one persona VM disk. The guest rejects any manifest that is unsigned or
// older than the last version it has seen.
type Manifest struct {
	AccountID uint64
	Version   uint64 // strictly increasing
	Entries   []ManifestEntry
	MerkleRoot [32]byte
	Signature  []byte // Ed25519 signature over (AccountID, Version, MerkleRoot)
	CreatedAt  time.Time
}

// SessionID identifies one guest session on a borrowed phone. Codes rotate
// after every session (see "Unlock flow" in docs/architecture.md).
type SessionID [16]byte

// AccountCode is the short, rotating code a user types on a borrowed phone
// to identify their account. Never used directly as key material.
type AccountCode string
