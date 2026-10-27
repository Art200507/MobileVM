// Command lease-server is unavailable. A future lease may fence server writes;
// it cannot force an offline device to stop or forget previously released keys.
package main

import "log"

func main() {
	log.Fatal("lease-server: not implemented yet (Phase 5, see docs/architecture.md)")
}
