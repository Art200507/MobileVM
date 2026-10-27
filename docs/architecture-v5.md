# ShadowDevice: Security Architecture (v5)

> **Review status (2026-10-21):** this is the historical v5 proposal, not a
> validated security specification. Read [design-review.md](design-review.md)
> and [phase-1-plan.md](phase-1-plan.md) first. They supersede conflicting claims
> below: ordinary third-party AVF access is restricted; Terminal does not prove
> protected Waydroid support; Waydroid can use software rendering; no device list
> has been validated for ShadowDevice; signed versions alone do not guarantee
> freshness on a new device; and PIN recovery on an ordinary server does not
> provide breach-resistant guess limits. Protected memory does not secure
> host-controlled input/display or guarantee remote erasure.

Condensed from `ShadowDevice Security Architecture.pdf` for quick reference during
implementation. See the PDF for full detail, threat tables, and sources.

## Idea

Your "phone" lives inside a VM from day one. The VM's disk syncs to the cloud as
encrypted blocks. Any supported Android phone can stream and decrypt it after you
prove who you are: enter a short code + PIN, get your persona running inside an app
on someone else's phone, and leave nothing behind when you close it.

## Four core rules

1. **The cloud never sees plaintext.** It stores encrypted, fixed-size blocks only.
2. **The code never is the key.** A random 256-bit master key (MK) encrypts
   everything. Code + PIN only unlock it via a guess-limited recovery service.
3. **Stream, don't download.** The borrowed phone fetches only the blocks it touches.
4. **Assume the borrowed phone is hostile.** Decrypted data stays in protected VM
   memory, never on disk; sessions are time-limited, one-time, remotely killable.

## Components (3 zones)

- **Your phone:** hardware keystore (MK + signing key), Persona VM, sync agent.
- **Cloud (ciphertext only):** recovery service, lease server, block store, signed
  manifest.
- **Borrowed phone:** ShadowDevice app (attested), protected VM (pKVM, RAM-only
  decrypt), wiped cache on session end.

## Keys

| Key | Derived from | Protects | Lives where |
|---|---|---|---|
| Master key (MK) | 256 random bits, generated on your phone | everything | hardware keystore; wrapped copies in recovery service |
| Block key | HKDF(MK, "blocks") | block contents | memory only |
| Block-ID key | HKDF(MK, "ids") | block names (anti-dedup-leak) | memory only |
| Signing key | Ed25519, generated on your phone | manifest | hardware keystore; pubkey registered with server |
| Session cache key | random per session | blocks cached on borrowed phone | protected VM memory only, destroyed at session end |

## Block encryption

- Fixed 4 MB blocks, XChaCha20-Poly1305 (or AES-256-GCM), fresh random nonce.
- AAD includes block index + snapshot version (prevents swap/replay).
- Manifest: Merkle tree over block IDs, signed, monotonic version counter.

## Unlock flow (borrowed phone)

1. Enter account code (rotates each session) + PIN.
2. OPAQUE login (RFC 9807) — server never sees the PIN.
3. Argon2id-stretch PIN → ask recovery service for secret share (~10 guess limit,
   Signal-SVR-style; SVR3 split-trust model for production).
4. PIN-derived key + share unwrap MK inside the protected VM only.

## Compatibility

| Phone | Mode |
|---|---|
| Pixel 6+ (Tensor), Android 16+ | Full VM |
| Samsung Galaxy S26/S26+ (Exynos) | Full VM, reported/unverified |
| Snapdragon Android | Essentials Vault |
| iPhone | Essentials Vault (no JIT, apps sandboxed) |
| Any browser | Essentials Vault (web) |

Full VM runs Android apps via Waydroid inside the AVF Debian guest — **unproven,
least-validated part of the design.** Banking apps fail Play Integrity inside a VM.
No SIM, so SMS 2FA needs an authenticator app or vault backup codes.

## Build plan (phase gates)

0. **Feasibility spike** — Waydroid inside AVF Debian on a real Pixel. Gate: one
   Android app opens inside the VM. *(needs physical hardware)*
1. **Encrypted streaming (core MVP)** — split a Linux VM disk into encrypted 4 MB
   blocks on a server; a second device boots it by fetching blocks on demand. Gate:
   boots without full download; server only ever holds ciphertext.
2. **Key recovery** — OPAQUE login, Argon2id, server-side guess limit, printed
   recovery kit. Gate: 10 wrong PINs locks the share; kit still recovers.
3. **Hostile-guest hardening** — protected VM, RAM-only crypto-shredded cache, app
   attestation, one-time codes, session timer, remote kill. Gate: nothing readable
   left on the guest after a session.
4. **Essentials Vault** — contacts/notes/documents/backup codes, any phone/browser.
   Gate: works on an iPhone.
5. **Owner sync** — background delta sync, single-active lease, duress PIN. Gate: a
   day of normal use syncs within a data budget.

Phases 1–3 alone are a strong standalone deliverable. Phase 0 decides whether the
full "apps on any phone" version is realistic.

## Open risks

- Protected VMs (memory-hidden) vs. GPU-capable VMs (needed for Waydroid) may be
  mutually exclusive on current hardware.
- Waydroid-in-AVF is unproven on phones.
- Full-VM device coverage is currently a minority of phones.
- Screen capture on a compromised guest is not solvable in software.
- First full sync of a large persona VM is slow/costly — keep it tens of GB, not 256.
