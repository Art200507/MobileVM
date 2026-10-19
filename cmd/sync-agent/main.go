// Command sync-agent runs on the owner's phone, inside or alongside the
// Persona VM. It snapshots the VM disk, encrypts changed blocks, builds and
// signs a new manifest, and uploads both to the blockstore-server. Phase 1
// (core MVP) for one-shot sync; Phase 5 (owner sync) for background delta
// sync on Wi-Fi while charging, within a data budget.
//
// Planned responsibilities (see docs/architecture.md):
//   - Take a copy-on-write snapshot of the Persona VM disk.
//   - Diff against the last synced snapshot to find changed blocks only.
//   - Encrypt changed blocks (internal/block) and upload them.
//   - Build + sign a new manifest (internal/manifest) with a bumped version
//     and upload it last, so partial uploads never leave a guest able to
//     boot an inconsistent disk.
package main

import "log"

func main() {
	log.Fatal("sync-agent: not implemented yet (Phase 1/5, see docs/architecture.md)")
}
