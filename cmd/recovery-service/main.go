// Command recovery-service is the cloud-side guess-limited key recovery
// server: OPAQUE registration/login so it never sees the PIN, Argon2id
// verification, and a per-account guess counter that destroys the secret
// share after MaxGuesses wrong attempts. Phase 2.
//
// For an MVP this is a single server with rate limiting. A real product
// should split the share across independently-operated enclaves
// (Signal-SVR3-style split trust) so no single compromised server or admin
// can lift the guess limit — see "Making the guess limit trustworthy" in
// docs/architecture.md.
package main

import "log"

func main() {
	log.Fatal("recovery-service: not implemented yet (Phase 2, see docs/architecture.md)")
}
