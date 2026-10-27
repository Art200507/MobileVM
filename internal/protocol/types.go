// Package protocol defines the versioned development disk format.
package protocol

const (
	FormatVersion uint32 = 1
	BlockSize            = 4 << 20 // 4 MiB of plaintext, plus sealed-object overhead.
	MaxBlocks            = 65536   // Bound metadata allocation; at most 256 GiB per image.
	Suite                = "AES-256-GCM-HKDF-SHA256"
)

type DiskID [16]byte
type BlockID [32]byte
type Digest [32]byte

// Context identifies an immutable block generation, not its referencing snapshot.
type Context struct {
	DiskID     DiskID
	KeyEpoch   uint64
	Index      uint64
	Generation uint64
}

// EncryptedBlock has a fresh 256-bit salt for a per-object encryption subkey.
// Sealed contains a library-generated nonce followed by ciphertext and tag.
type EncryptedBlock struct {
	Format  uint32
	Context Context
	Salt    [32]byte
	Sealed  []byte
}
type ManifestEntry struct {
	Index      uint64
	Generation uint64
	BlockID    BlockID
}

// Manifest authenticates geometry, history and entry semantics. No wall clock is
// used for ordering. The signer must be provisioned independently of storage.
type Manifest struct {
	Format     uint32
	Suite      string
	DiskID     DiskID
	KeyEpoch   uint64
	Version    uint64
	Parent     Digest
	DiskBytes  uint64
	BlockBytes uint32
	Entries    []ManifestEntry
	MerkleRoot [32]byte
	Signature  []byte
}
type SessionID [16]byte
type AccountCode string
