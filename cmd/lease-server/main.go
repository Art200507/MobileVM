// Command lease-server ensures only one copy of a persona runs at a time:
// starting a new guest session acquires the account's lease and revokes any
// session on the owner's lost phone. Phase 5 (owner sync / single-active
// lease), used by session.Start from Phase 3 onward.
package main

import "log"

func main() {
	log.Fatal("lease-server: not implemented yet (Phase 5, see docs/architecture.md)")
}
