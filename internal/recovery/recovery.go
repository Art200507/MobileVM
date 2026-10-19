// Package recovery implements the guess-limited key recovery flow: OPAQUE
// login (RFC 9807) so the server never sees the PIN, Argon2id PIN
// stretching, and a server-side guess counter that destroys the secret
// share after ~10 wrong attempts. Phase 2. See "Making the guess limit
// trustworthy" in docs/architecture.md for the production split-trust
// (SVR3-style) direction beyond a single-server MVP.
package recovery

import (
	"errors"

	"shadowdevice/internal/keys"
)

var errNotImplemented = errors.New("recovery: not implemented yet (see docs/architecture.md Build plan)")

// MaxGuesses is the fixed number of PIN attempts allowed before the
// recovery service destroys the account's secret share.
const MaxGuesses = 10

// Share is one secret share that, combined with a PIN-derived key, unwraps
// the master key. Held by the recovery service, released only on a correct
// OPAQUE login and within the guess limit.
type Share []byte

// Register performs the OPAQUE registration (sign-up) for a new PIN,
// without the server ever learning the PIN itself.
func Register(accountCode, pin string) error {
	return errNotImplemented
}

// Unlock performs an OPAQUE login against the recovery service, and on
// success returns the share needed (together with the Argon2id-stretched
// PIN) to unwrap the master key inside the protected VM. A wrong PIN
// consumes one of MaxGuesses attempts.
func Unlock(accountCode, pin string) (Share, error) {
	return nil, errNotImplemented
}

// UnwrapMasterKey combines a PIN-derived key and the released share to
// recover the master key, inside the protected VM only — it must never
// touch the borrowed phone's normal Android.
func UnwrapMasterKey(pin string, share Share) (keys.MasterKey, error) {
	return keys.MasterKey{}, errNotImplemented
}

// RecoveryKit represents the printed 24-word code (and optional 2-of-3
// Shamir split among trusted contacts) that wraps the master key
// independently of the PIN, for when the PIN is forgotten or guesses run
// out.
type RecoveryKit struct {
	Words []string
}

// GenerateRecoveryKit produces a new recovery kit wrapping mk.
func GenerateRecoveryKit(mk keys.MasterKey) (RecoveryKit, error) {
	return RecoveryKit{}, errNotImplemented
}

// RecoverFromKit unwraps the master key from a recovery kit.
func RecoverFromKit(kit RecoveryKit) (keys.MasterKey, error) {
	return keys.MasterKey{}, errNotImplemented
}
