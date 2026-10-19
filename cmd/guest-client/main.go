// Command guest-client runs inside the protected VM on a borrowed phone. It
// unlocks the owner's master key via the recovery service, verifies the
// latest manifest, and lazily streams + decrypts only the blocks the
// running persona touches. Phase 1 (lazy streaming) through Phase 3
// (hostile-guest hardening).
//
// Planned responsibilities (see docs/architecture.md):
//   - Collect account code + PIN, run the OPAQUE unlock flow
//     (internal/recovery) to obtain the master key, entirely inside the
//     protected VM.
//   - Fetch and verify the signed manifest (internal/manifest); reject
//     unsigned or stale versions.
//   - Serve a block device backed by on-demand fetch + decrypt
//     (internal/block) from blockstore-server, backed by a prefetched boot
//     set for fast start.
//   - Enforce the session timer and respond to remote kill
//     (internal/session) by crypto-shredding the session cache key.
package main

import "log"

func main() {
	log.Fatal("guest-client: not implemented yet (Phase 1-3, see docs/architecture.md)")
}
