// Command blockstore-server is the cloud-side server that stores encrypted
// 4 MB blocks and signed manifests, and serves them to guest clients on
// demand. It never holds plaintext or any key material. Phase 1 (core MVP).
//
// Planned responsibilities (see docs/architecture.md):
//   - PUT block: accept an EncryptedBlock keyed by its BlockID, reject
//     overwrites of an existing ID (blocks are content-addressed).
//   - PUT manifest: accept a signed Manifest, reject it if Version is not
//     strictly greater than the last stored version for the account.
//   - GET block/{id}: serve ciphertext for a single block.
//   - GET manifest/{accountID}: serve the latest signed manifest.
package main

import "log"

func main() {
	log.Fatal("blockstore-server: not implemented yet (Phase 1, see docs/architecture.md)")
}
