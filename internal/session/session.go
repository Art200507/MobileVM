// Package session manages guest sessions on borrowed phones: app
// attestation before any key is released, one-time session codes, the
// session timer, and remote kill — plus the single-active lease that
// revokes the owner's lost phone when a new session starts. Phase 3 (guest
// hardening) and Phase 5 (single-active lease). See docs/architecture.md.
package session

import (
	"errors"
	"time"

	"shadowdevice/internal/protocol"
)

var errNotImplemented = errors.New("session: not implemented yet (see docs/architecture.md Build plan)")

// DefaultTimeout bounds how long a guest session may run before it is
// force-closed, limiting the window for screen capture on a hostile guest.
const DefaultTimeout = 30 * time.Minute

// State tracks one active guest session.
type State struct {
	ID        protocol.SessionID
	AccountID uint64
	StartedAt time.Time
	ExpiresAt time.Time
}

// Attest verifies the ShadowDevice app on the borrowed phone is genuine
// before any key material is released to it.
func Attest(appAttestationToken []byte) error {
	return errNotImplemented
}

// Start begins a new session: acquires the single-active lease (revoking
// any session on the owner's other devices), issues a one-time session
// code, and starts the session timer.
func Start(accountID uint64) (State, error) {
	return State{}, errNotImplemented
}

// Kill remotely and immediately ends a session, instructing the guest to
// crypto-shred its session cache key.
func Kill(id protocol.SessionID) error {
	return errNotImplemented
}

// Close ends a session normally at the owner's request or on timeout.
func Close(id protocol.SessionID) error {
	return errNotImplemented
}
