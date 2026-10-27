// Package session reserves interfaces for future protected sessions. It is not
// implemented: app attestation alone is insufficient, and revocation cannot
// erase data or stop an offline hostile host. See docs/design-review.md.
package session

import (
	"errors"
	"time"

	"shadowdevice/internal/protocol"
)

var errNotImplemented = errors.New("session: not implemented yet (see docs/architecture.md Build plan)")

// DefaultTimeout is a proposed cooperative timeout, not a security guarantee.
const DefaultTimeout = 30 * time.Minute

// State tracks one active guest session.
type State struct {
	ID        protocol.SessionID
	AccountID uint64
	StartedAt time.Time
	ExpiresAt time.Time
}

// Attest is unavailable. Production verification must measure the pVM payload
// and bind secret delivery to its attested key, not just check an app token.
func Attest(appAttestationToken []byte) error {
	return errNotImplemented
}

// Start is unavailable pending authenticated authorization, delivery-channel
// binding, fencing, lifecycle and trusted input/output decisions.
func Start(accountID uint64) (State, error) {
	return State{}, errNotImplemented
}

// Kill is unavailable. A future version may revoke server capabilities and
// request cooperative shutdown; immediate offline erasure cannot be promised.
func Kill(id protocol.SessionID) error {
	return errNotImplemented
}

// Close ends a session normally at the owner's request or on timeout.
func Close(id protocol.SessionID) error {
	return errNotImplemented
}
