// Package manifest builds, signs, and verifies the Merkle manifest that
// lists every block in a persona VM disk snapshot, with a strictly
// increasing version counter. Phase 1 (core MVP).
package manifest

import (
	"crypto/ed25519"
	"errors"

	"shadowdevice/internal/protocol"
)

var errNotImplemented = errors.New("manifest: not implemented yet (see docs/architecture.md Build plan)")

// Build computes the Merkle root over the given block entries and returns
// an unsigned manifest at the given version.
func Build(accountID, version uint64, entries []protocol.ManifestEntry) (protocol.Manifest, error) {
	return protocol.Manifest{}, errNotImplemented
}

// Sign signs a manifest's (AccountID, Version, MerkleRoot) with the owner's
// Ed25519 signing key.
func Sign(priv ed25519.PrivateKey, m protocol.Manifest) (protocol.Manifest, error) {
	return protocol.Manifest{}, errNotImplemented
}

// Verify checks a manifest's signature against the owner's registered
// public key, and that its version is strictly greater than lastSeenVersion
// (rejecting unsigned or stale manifests, per the "borrowed phone rejects
// any manifest that is unsigned or older" rule).
func Verify(pub ed25519.PublicKey, m protocol.Manifest, lastSeenVersion uint64) error {
	return errNotImplemented
}
