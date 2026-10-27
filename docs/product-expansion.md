# Product expansion: selective extraction, compression, storage, and AI

Status: proposals for discussion, 2026-10-27. None of this is implemented
except where explicitly noted. See [architecture.md](architecture.md) for
what's actually built and [design-review.md](design-review.md) for the trust
model these proposals must not violate.

## 1. Selective data extraction ("what gets pulled off the phone")

The ask: let a user choose, per category (Photos, Messages, Contacts,
Documents, a specific app's data...), what gets included in their persona at
all — not just what's visible in a reduced-privilege mode, but what's ever
encrypted and uploaded in the first place.

**This does not need a new primitive.** Look at the key derivation already
built: `DeriveBlockKey(mk, disk, epoch)` and `DeriveBlockIDKey(mk, disk,
epoch)` both take a `protocol.DiskID` as input, and `image.Publish` already
treats disk identity as a first-class parameter, not a constant. The system
already supports many independently-keyed, independently-synced,
independently-pinned "disks" under one master key — it's just only ever been
used for one disk at a time.

Selective extraction is a product/app-layer decision on top of that, not a
crypto or format change:

- Each selectable category becomes its own `DiskID` (e.g., derived
  deterministically as `HKDF(MK, "disk:photos")`, `HKDF(MK, "disk:messages")`,
  etc., so the app can regenerate the same ID without storing a lookup table).
- Each category gets its own manifest chain, versioned and synced
  independently — meaning "re-sync just Photos" or "stop syncing Messages
  entirely" are both free operations, not a reindex of one monolithic image.
- A category a user opts out of simply never has `Publish` called for it.
  Nothing is uploaded, not even as a size signal — there's no manifest, no
  blocks, no trace, because the whole pipeline is per-disk already.
- Recovery/sharing scope follows naturally: the Essentials Vault (#4 in the
  architecture doc) is just the subset of category-disks a user marks as
  "available in reduced mode," each still independently keyed — which also
  directly answers an open item from the design review (#21: "no
  master-persona key in an essentials-only browser client"). A browser
  client only ever needs the keys for the categories it's allowed to touch.

What this *doesn't* solve: the UI/UX of letting a user actually choose
("extract my Photos but not my Camera Roll screenshots" is a much finer
granularity question), and the fact that a full VM persona — the original
"run my apps" mode — is one filesystem, not naturally decomposable into
clean categories the way a Vault-style structured dataset is. Treat these as
two different products sharing one key-derivation scheme: a **VM-persona
disk** (opaque, monolithic, for Android-app execution) and a set of
**category datasets** (structured, independently selectable, for Essentials
Vault and granular opt-in/opt-out). Don't try to force the VM-persona disk
itself to be selectively prunable at the file level — that's a much harder
problem (equivalent to doing selective restore from inside someone else's
filesystem) and isn't needed for the stated goal.

## 2. Compression

### The constraint that makes this non-trivial

`docs/architecture.md`'s fixed-block design exists specifically so block
size can't leak which files exist. Compressing before encrypting and then
storing the compressed size changes the ciphertext's length as a function of
plaintext content — reintroducing exactly the leak fixed sizes were built to
prevent (the classic CRIME/BREACH category of issue, here applied to a
disk-image block store instead of HTTP).

**Compress, then pad to a fixed size — never store the compressed length.**
Concretely:

1. Compress the plaintext block.
2. If the compressed result is smaller than the block size, pad it
   (zero-fill is fine; after AEAD sealing it's indistinguishable from any
   other padding) back up before encrypting.
3. The decompressor reads until it finds the compression stream's own
   end-of-stream marker and ignores the padded tail — most stream formats
   (including `compress/flate`) are self-terminating, so this needs no extra
   length field in the authenticated header.

Doing *only* step 1–3 against a single fixed block size, though, captures
**none** of the storage/bandwidth benefit — the stored object is still
exactly the same size either way, so a highly-compressible (sparse, mostly
zero, or text-heavy) block costs exactly as much to store and transfer as an
incompressible one. The entire point of compressing is lost unless sizes are
allowed to vary *somewhat*.

**The actual mechanism: a small number of fixed size tiers, not one size.**
E.g. 256 KiB / 1 MiB / 2 MiB / 4 MiB. A block's sealed size reveals which
*tier* it landed in — a coarser, deliberately-accepted leak (roughly "mostly
empty" vs. "dense") in exchange for real savings on the sparse regions that
dominate a typical VM disk's unused space, and on text/log/database content.
This tradeoff needs to be an explicit, documented decision — not an
accident — exactly in the spirit of `design-review.md`'s treatment of
access-pattern leakage as "accepted metadata leakage, declared" rather than
silently ignored.

Two things this does *not* help much with, so expectations should be
calibrated: media that's already compressed at the format level (photos,
video, most document formats) gains little to nothing from a second pass —
the benefit concentrates on sparse disk regions, logs, databases, and plain
text. And the manifest/block-count format currently assumes one fixed
`BlockSize` throughout (`manifest.validate` rejects anything else) — adding
tiers is a real wire-format change, not a drop-in.

### Library choice

The project has deliberately stayed stdlib-only so far (see README). zstd
would give a better ratio/speed tradeoff than anything in the standard
library, but isn't in it — adding it means a real third-party dependency,
which is itself a supply-chain decision worth making consciously (concern
#23 from the original design review was exactly "supply-chain risk in the
app itself"). `compress/flate` is a reasonable stdlib-only starting point:
not as good as zstd, but good enough to validate the whole compress-then-pad
mechanism without taking on a new dependency. Swapping the compressor later
is a localized, low-risk change if the dependency tradeoff is later decided
differently — nothing else in the design depends on which algorithm is used.

**Implemented this session**: a standalone, stdlib-only `internal/compress`
package (flate-based, bounded-output decompression, fuzz-tested) — see
`internal/compress/compress.go`. It is **not yet wired into
`internal/block`'s fixed-size pipeline**; doing that requires the tiered-size
format decision above, which changes the authenticated wire format and
should be made deliberately, not as a side effect of adding a utility
function. Treat the package as a proven building block, not a shipped
feature.

## 3. Cloud / storage backend

Nothing about the trust model depends on *which* cloud: the whole point of
client-side encryption is that the storage backend only ever sees ciphertext
blobs addressed by opaque keyed IDs. This makes the backend choice a
cost/ops decision, not a security one — pick on economics and operational
fit, not on "which cloud is most secure," because none of them ever hold a
key.

The access pattern that matters here: ShadowDevice's core design is
**lazy, on-demand block fetch** — a borrowed-phone session, by design, reads
a small fraction of total stored bytes, and the earlier design review
flagged "upload cost" and "egress" as open risks. That makes egress pricing,
not storage pricing, the number worth optimizing. **Cloudflare R2** (S3-API
compatible, so no application-level lock-in beyond config) currently prices
storage at $0.015/GB/month with **zero egress fees** — for a workload that's
occasionally-read, heavily fetched in bursts (exactly a recovery session's
shape), that's a materially different cost curve than S3's per-GB egress
charges, which could otherwise dominate the bill precisely during the
highest-value moment (someone actually using the product in an emergency).
Recommendation: default to an S3-compatible backend so the choice isn't
locked in, but start on R2 or similar zero/low-egress storage specifically
*because* of this access pattern, not generic "cheaper storage."

Multi-provider replication (mentioned in the original PDF's recovery
section) is a legitimate later hardening step for availability — but adds
real operational cost and should follow, not precede, validating the product
on one backend.

## 4. Encryption depth

Current state (`internal/block`, `internal/keys`) is already solid: AES-256
with a fresh HKDF-derived subkey per object (so no key ever encrypts more
than one message — the standard GCM nonce-reuse risk is structurally
eliminated, not just mitigated by randomization) plus a random GCM nonce on
top. 256-bit keys throughout. This isn't a place that needs more bits; more
bits wouldn't change anything about the real risks in `design-review.md`
(host compromise, recovery server trust, screen capture) — those are
boundary/trust problems, not key-length problems.

The one forward-looking gap worth naming: **harvest-now-decrypt-later**.
Data encrypted today and stored for years is exactly the scenario
post-quantum migration guidance targets, and NIST finalized ML-KEM (FIPS
203) in August 2024 — it's a settled standard now, not a research proposal.
Production deployments already exist: Akamai began shipping hybrid
X25519+ML-KEM key exchange in September 2025, moving to default for all
customers in February 2026. The pattern worth copying for ShadowDevice:
**hybrid, not a replacement** — wrap the master key (or a key-wrapping layer
above it, in the recovery service once that's built) with both classical
(X25519/Ed25519, already in use) and ML-KEM, so security depends on
*neither* primitive alone being broken. This is a Phase 2/recovery-era
decision, not something to bolt onto the current block format, since it's
the key *recovery and wrapping* layer — not disk content encryption — where
long-lived exposure actually accumulates.

## 5. Data form

Two different "forms" are being conflated in the current single-disk model,
and separating them cleanly matters for both selective extraction (#1) and
for what's realistic on iOS (`ios-platform-research.md`):

- **Opaque block form** (what exists today): a raw disk image, chunked into
  fixed blocks, meaningful only once reassembled and mounted by a VM.
  Required for the "run my actual apps" persona mode. Not selectively
  prunable, not individually inspectable, not useful outside a VM context.
- **Structured record form** (not yet built): individually-encrypted,
  individually-addressable items — a contact, a document, a photo — each
  with its own authenticated metadata (filename, type, size class) rather
  than being indistinguishable bytes inside a disk image. This is what
  Essentials Vault and selective extraction actually want: something a
  browser or App Clip can decrypt and show *one item at a time* without
  mounting a filesystem at all.

These aren't competing designs, they're complementary and already compatible
with the per-category-disk model in #1: a structured dataset is just a
disk-scoped manifest whose "blocks" are individually-sized encrypted
records instead of 4 MiB disk chunks. Building this is a real scope of work
(a new entry format, size-class bucketing per #2's reasoning, a different
reader shape) but doesn't require new cryptographic primitives — same
derivation hierarchy, same AEAD, same signed-manifest pattern, applied to
smaller independent objects instead of disk blocks.

## 6. Where AI fits — and a hard boundary on where it must not

Concrete, scoped roles that fit this architecture without weakening it:

- **On-device categorization for the extraction picker (#1).** A small
  on-device model scans Photos/Messages/Documents locally and proposes
  category groupings ("128 screenshots," "47 photos with faces," "3 years of
  one conversation thread") so the opt-in/opt-out picker in #1 is answering
  real questions instead of raw folder names. Runs entirely client-side,
  before anything is encrypted — model output never leaves the device unless
  the user acts on it by changing what gets synced.
- **Smarter delta sync.** The current incremental-upload logic reuses a
  block only on exact byte match. Perceptual near-duplicate detection (e.g.
  for burst-mode photos, or lightly-edited versions of the same image) could
  catch "this is 99% the same photo" cases that byte-level diffing misses —
  still a local, pre-encryption computation, same trust boundary as today.
  This is an efficiency feature, not a trust-model change; expect real
  engineering (the current block-diff path is `bytes.Equal`, nothing more).
- **Server-side anomaly detection on access metadata only.** The design
  review already accepts that the server sees block-fetch counts, timing,
  and sizes (concern #12/#13, "accepted metadata leakage, declared"). That
  same visible metadata is enough to build real signal — e.g., "this
  account's lease just fetched an unusual volume of blocks from a new
  region" — without the server ever needing plaintext. This is the one
  place "AI on the server" is safe by construction, because it's operating
  on data the trust model already discloses.
- **In-session local assistance during an actual emergency.** Inside a
  borrowed-phone Essentials session, content is already decrypted in that
  session's trusted boundary (whatever that boundary ends up being — see
  `design-review.md`'s unresolved endpoint-trust questions). A local search/
  assistant over that already-decrypted data ("find my passport number," "my
  last boarding pass") is not a new trust boundary, just a better interface
  to data that's already there. Still has to respect the screen/input trust
  limits already documented — an AI assistant doesn't change what a hostile
  host can observe.

**The hard boundary: no plaintext content, ever, to a third-party or
cloud-hosted AI API.** This is worth stating explicitly because it's the
easiest mistake to make by accident — "auto-tag photos" or "summarize my
messages" sound like natural AI features, but routing plaintext through an
external inference API is a direct, silent violation of the one rule this
entire project's trust story depends on ("the cloud never sees plaintext").
Any AI feature here has exactly two acceptable execution boundaries:
on-device/client-side before encryption, or inside the same protected
execution boundary already defined for the persona itself. If a proposed AI
feature can't be scoped to run in one of those two places, it doesn't belong
in this architecture — build it as a separate, clearly-labeled opt-in
feature with its own explicit consent and threat-model disclosure, never as
a default.
