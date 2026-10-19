// Package block implements fixed-size block encryption for the persona VM
// disk: splitting a disk image into BlockSize chunks, sealing each with an
// AEAD cipher under the block key, and computing keyed block IDs under the
// block-ID key. Phase 1 (core MVP).
package block

import (
	"errors"

	"shadowdevice/internal/keys"
	"shadowdevice/internal/protocol"
)

var errNotImplemented = errors.New("block: not implemented yet (see docs/architecture.md Build plan)")

// Seal encrypts one plaintext block (must be exactly protocol.BlockSize,
// zero-padded if shorter) under the block key, using a fresh random nonce
// and binding index + snapshotVersion as associated data so the server
// can't swap or replay blocks.
func Seal(bk keys.BlockKey, index, snapshotVersion uint64, plaintext []byte) (protocol.EncryptedBlock, error) {
	return protocol.EncryptedBlock{}, errNotImplemented
}

// Open decrypts and authenticates a block, verifying its bound index and
// snapshot version match what the caller expects.
func Open(bk keys.BlockKey, eb protocol.EncryptedBlock) (plaintext []byte, err error) {
	return nil, errNotImplemented
}

// DeriveID computes the keyed block identifier for a block's ciphertext
// under the block-ID key, so identical plaintext blocks don't leak as
// identical IDs to anyone without the key.
func DeriveID(idk keys.BlockIDKey, index uint64, ciphertext []byte) (protocol.BlockID, error) {
	return protocol.BlockID{}, errNotImplemented
}

// Split divides a raw disk image into fixed-size plaintext chunks, padding
// the final chunk to protocol.BlockSize.
func Split(diskImage []byte) [][]byte {
	return nil
}
