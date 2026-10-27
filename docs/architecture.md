# ShadowDevice architecture

Status: development prototype, 2026-10-25. This document describes implemented
behavior. [architecture-v5.md](architecture-v5.md) preserves the historical
proposal. The [design review](design-review.md) covers its unsupported claims.

## Implemented trust boundary

The sync process and reader process run on trusted development machines. They
handle plaintext and portable secret keys in ordinary Go process memory. The
filesystem holding their identity files is trusted. The storage server receives
sealed blocks and signed manifests, not plaintext or secret keys.

A reader must possess a separately provisioned reader identity (disk ID, data root
and signing public key) and an independently trusted manifest digest. Neither
identity nor checkpoint is obtained from the ciphertext server. This prevents
substitution of the expected snapshot. It does not discover the latest snapshot
for a fresh device. Reading an explicitly pinned old snapshot is allowed.

## Disk format v1

- Blocks contain 4 MiB of padded plaintext. Geometry, including exact byte length,
  is signed and visible; access patterns and object reuse remain visible.
- HKDF-SHA256 derives separate disk/epoch encryption and object-ID keys from a
  portable random 256-bit data root. The signing seed is independent.
- Each sealed block has a fresh 256-bit salt deriving a per-object AES-256 key.
  Go's AES-GCM random-nonce API prepends its 96-bit nonce and appends the tag.
  Each derived object key seals one message; reusing an object does not reseal it.
- AEAD binds format, disk ID, epoch, logical index, generation and salt. Generation
  records creation, rather than every subsequent snapshot that references it.
- HMAC-SHA256 object IDs cover the complete authenticated header and sealed body.
  Readers compare object context and ID against a verified manifest entry.
- Ed25519 signatures cover format/suite, disk ID, epoch, version, parent hash,
  exact disk length, block size, entry count and ordered Merkle root. Leaves bind
  index/generation/object ID; leaf and parent hashes have different prefixes.
  Odd Merkle levels duplicate the last node; an empty image has a distinct root.
- All integer encodings used for cryptography are fixed-width big endian; suite
  text has a 32-bit length prefix. Manifest JSON is transport only. Object binary
  fields are format:u32, disk:16 bytes, epoch:u64, index:u64, generation:u64,
  salt:32 bytes, and sealed body:4 MiB+28 bytes. Object size is 4,194,408 bytes.

AES was selected from the original proposal's supported alternatives to use Go's
standard cryptographic implementations without adding a dependency. This replaces
XChaCha20 as the proposed initial suite in the older plan. No cipher negotiation
or custom cryptographic primitives are implemented. Reference:
[Go random-nonce GCM](https://pkg.go.dev/crypto/cipher#NewGCMWithRandomNonce),
[Go HKDF](https://pkg.go.dev/crypto/hkdf).

## Snapshot publication

The writer takes a caller-supplied frozen/offline raw image, not a live VM. It
reads blocks incrementally and compares against authenticated prior blocks to
reuse unchanged objects. It does not provide plaintext-hash deduplication.

Immutable objects and manifests are written through synced temporary files and
published using non-overwriting links. After checking every referenced object is
present and structurally consistent, a process-shared file lock protects the
expected-parent check and head update. The head is replaced by rename and the
directory is synced. The store is single-disk/single-signing-authority; a stale
parent returns a conflict and does not replace the head. Failed uploads may leave
orphan objects. GC and multi-provider replication are not implemented.

Filesystem stat checks detect some changes during publication. They cannot prove
that an image was frozen or applications were quiesced. Local filesystem durability,
path ownership and supported fsync/locking behavior remain assumptions. The head
service is trusted for serialization; a malicious service could fork histories.

## On-demand access

The reader fetches only requested blocks, authenticates them, and retains at most
one decrypted block. It implements io.ReaderAt, including cross-block and final
partial reads. Network responses and metadata are bounded. A missing/corrupt block
is an error; it is never replaced with zeroes.

The ciphertext HTTP service is read-only and requires numeric loopback binding.
The reader can optionally expose plaintext single-range reads on a second
loopback endpoint for local VM tools. That endpoint authenticates the entire
requested range before replying, caps a request at 32 MiB, and denies writes.
It deliberately exposes plaintext to the trusted local host. It is not a protected
VM channel. No Linux VM boot or Android hardware experiment has been completed.

## Features deliberately unavailable

PIN/OPAQUE recovery, Android protected execution/attestation, hardware wrapping,
signer migration/rotation, leases/fencing, enforced expiration, duress mode,
Essentials Vault, writable remote sessions, background owner sync and GC.
Existing recovery/session interfaces return errors; they are not production paths.

Revocation cannot erase previously copied secrets or stop an offline device.
Clearing owned Go buffers does not prove erasure of runtime copies or host caches.
Hostile keyboard/display protection, endpoint supply-chain trust, app distribution,
MDM consent and production legal/economic decisions remain separate gates.
