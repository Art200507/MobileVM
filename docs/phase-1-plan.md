# Proposed Phase 1: authenticated encrypted disk streaming

Status: original design proposal dated 2026-10-21. Implementation began with
user authorization on 2026-10-25; see [architecture.md](architecture.md) for the
implemented AES-GCM per-object-subkey suite and [remaining-work.md](remaining-work.md)
for unpassed gates. This document preserves the earlier proposal. See [design-review.md](design-review.md) for evidence and risks.
Existing scaffold APIs require revision before implementing this plan.

## Scope and trust model

Demonstrate demand-paged access to an immutable encrypted Linux disk snapshot on
a trusted development machine. Use synthetic data, locally provisioned random
keys, and a pinned signer public key. No PIN recovery, production Android support,
hostile-host guarantee, sensor passthrough, duress mode, or live owner sync.

The storage service may tamper with, reorder, omit, or replay objects. Clients
must detect authenticated-content substitution. Service availability is not
guaranteed. A fresh client obtains its expected manifest digest out of band for
this phase; do not claim autonomous latest-state discovery against that service.

Use a read-only remote base disk with a disposable local development overlay if
boot requires writes. Explicitly state that overlay writes are not backed up.
Disable guest networking or use a test workload with controlled external effects.
The fetch/decrypt adapter sits on the trusted development client; this is not
evidence that Android host-side decryption is acceptable in protected mode.

## Format decisions required before code

### Keys and cipher

Use a portable random 256-bit data root and domain-separated derived keys for a
specific disk and key epoch. A device Keystore key may later wrap the root; it
is not the portable root itself. Keep manifest signing separate. Phase 1 uses a
local development signer, not an assertion of hardware-backed Ed25519 support.

Choose one AEAD suite rather than runtime negotiation in the first format.
Recommended starting point: XChaCha20-Poly1305 with a fresh cryptographic random
nonce for each seal, using a maintained implementation. Bound input lengths,
reject unknown format versions, and never release unauthenticated plaintext.
Finalize suite/encoding against library documentation before implementation.

### Blocks and generations

Specify 4 MiB plaintext blocks (4,194,304 bytes), padding the final block. Encrypt
and authenticate the true disk byte length in metadata or sign it if length
disclosure is accepted. Account for nonce/tag/header overhead in stored size.
Fixed blocks hide exact file sizes, not disk capacity or access patterns.

Bind AEAD associated data to a canonical tuple:

`format version, disk ID, key epoch, logical block index, block generation`.

Generation identifies the write that created this immutable block. It is not the
version of every later snapshot referencing it. A new manifest can reuse the
same ciphertext and generation for an unchanged block. This avoids re-encrypting
the entire disk on every commit. Ensure fresh nonces even on retries or branches.

An entry authenticates index, generation, key epoch and object ID. Derive object
IDs from the complete canonical sealed object under a disk-scoped ID key; include
header and nonce, not just ciphertext. Fresh random encryption means this is
object identity, not plaintext-content deduplication. Reusing the same object
remains observable. A keyed name does not provide access-pattern hiding.

The decryption API must take expectations from a verified manifest entry and
compare them against the object before returning plaintext. Do not trust an
object's own Index/SnapshotVersion fields as the expected values.

### Manifest and signer authentication

Specify a deterministic, length-delimited encoding with domain separation for
signatures and Merkle leaves/internal nodes. Authenticate all authoritative
fields: format/suite, account and disk identity, key epoch, disk length, block
size/count, snapshot version, parent manifest hash, and entry root. Authenticate
signer identity/authorization through a pinned trust anchor. Timestamps are
informational unless explicitly part of an authenticated policy.

Require ordered unique indices covering exactly the declared image; reject
duplicates, out-of-range entries, invalid lengths and unsupported encodings.
Specify odd-node and empty-tree handling. Merkle leaves must bind complete
entries, not an unordered collection of object IDs.

Separate validation into signature/content integrity, known-checkpoint rollback
checks, and latest-head discovery. Reading an already pinned manifest again is
valid; equal-version/different-digest is an error. A strictly-greater check on
every Verify call would reject normal reopening of the current snapshot.

### Publication and crash consistency

1. Snapshot/quiesce the source; record filesystem versus application consistency.
2. Upload immutable sealed objects idempotently. Validate that an existing object
   with the requested ID is exactly the expected object; never overwrite by ID.
3. Confirm the objects satisfy the service's documented durable-write contract.
4. Upload the signed immutable manifest under its digest.
5. Atomically compare-and-swap the current head from the expected parent digest
   to the new digest. A failure preserves the competing branch and reports conflict.

For the demo, the head service is trusted for serialization and durability, not
for cryptographic freshness to a brand-new client. The trusted expected manifest
digest is supplied separately. A malicious service can fork views or lie about
durability; signatures and CAS alone do not solve that. A later independent
checkpoint/witness design needs its own threat model and rollback tests.

A crash before head publication leaves the old snapshot current; publication
must never intentionally expose an incomplete image. Orphan uploads may remain
until a safe cleanup window. Initial implementation may disable GC and enforce
quotas; later GC must mark retained manifests and pin in-flight readers/uploads
before sweeping. Do not promise physical deletion across cloud backups.

## Revised phase gates

| Phase | Acceptance gate |
|---|---|
| 0: deployment and protected execution | Record exact shipping firmware and privileges; show permitted launch path, protected boot and payload verification. Treat rendering and usable performance as separate tests. Development-only access is labelled as such. |
| 1A: format and publication | Resolve the above wire-format decisions; demonstrate authenticated round trips, immutable object reuse, crash-safe head updates and explicit stale-parent conflicts. |
| 1B: lazy disk access | Boot/read a synthetic Linux image via an on-demand block adapter without a full pre-download; verify bytes against the source and report actual unique bytes fetched. |
| 2: recovery | Select reviewed recovery protocol/library; specify server-compromise assumptions, attempt accounting, exhaustion abuse, kit recovery, signer rotation and replacement-device enrollment. |
| 3: protected endpoint | Bind delivery to measured pVM key and approved non-debug payload; audit plaintext buffers, input/output trust, cache/key lifetime, expiry and revocation limits. |
| 4: essentials | Separate dataset/key scope; document browser trust and weaker cleanup guarantees. Demonstrate a specific emergency task. |
| 5: writable sessions and owner sync | Define block flush/FUA semantics, durable overlays, fencing, conflict retention, application consistency, re-enrollment and tested recovery from interrupted writes. |

Fencing belongs to the first multi-writer design, not a cosmetic later lease.
The authoritative commit service must reject a stale writer epoch atomically
with the head change. Reader sessions need not be exclusive. Server-controlled
lease expiry and monotonic local timeout measurements have different roles;
neither wall-clock timestamps nor a lease can force an offline device to stop.

## Meaningful verification for implementation

- Corrupt nonce, header, ciphertext or tag; change expected disk/index/generation,
  object ID, signature or signer: reject without returning plaintext.
- Change disk geometry or manifest ordering; supply missing/duplicate entries or
  oversized metadata: reject with bounded resource usage.
- Reuse an unchanged block across two manifests; both snapshots decrypt correctly
  without uploading that block twice. Modified blocks use fresh seals.
- Simulate crashes after each publication step; a trusted head reader sees a
  complete old or new snapshot. Competing children cannot silently replace one
  another under an honest CAS authority.
- Serve an older valid manifest against a pinned checkpoint: reject. Also show
  that an uncheckpointed fresh client cannot detect that rollback unaided.
- Inject missing blocks and network failures; report errors, never silently
  substitute zeroes. Verify partial reads, final padding and image boundaries.
- Collect cold-boot bytes, unique blocks, request counts, cache use, startup time
  and memory. Report prefetch separately; fetching less than the image is a
  demonstration result, not a universal guarantee for every workload.

Block-device integration must be selected for the development host (for example,
a Linux NBD/ublk adapter inside a development VM). macOS package tests alone do
not satisfy the Linux boot gate. Document required tools and privileges before
starting that experiment.
