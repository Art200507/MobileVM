# Implementation status and remaining gates

Updated 2026-10-25. “Fixed” below refers to implemented prototype behavior, not
production readiness. The research review still records all 35 original issues.

## Implemented fixes

| Finding | Result |
|---|---|
| Block generation versus snapshot version | Immutable generations allow authenticated block reuse across snapshots. |
| Block verifier trusting supplied fields | Open requires expected disk/epoch/index/generation and object ID from a verified manifest. |
| Unsigned disk geometry and entry semantics | Canonical signed header plus ordered, domain-separated Merkle leaves. |
| Signer/bootstrap substitution | Reader identity pins signer and disk independently; exact manifest digest is required. |
| Partial upload and stale-parent overwrite | Immutable synced objects, complete-reference checks, filesystem lock and head CAS. |
| Cross-process competing commits | Only one child can become head under the trusted local store. |
| Silent corruption/missing objects | Read errors; no unauthenticated plaintext or silent zero-fill. |
| Whole-image download on read | ReaderAt fetches touched blocks only; one-block cache and fetch counters. |
| Unchanged-block upload cost | Compare with authenticated previous blocks and reuse identical objects. |
| Excessive transport/input size | Fixed object sizes, image/manifest/range bounds, timeouts and no remote redirects. |
| Reader carrying signing authority | Separate reader bundle; guest CLI rejects owner signing bundle. |
| Secret files/plaintext outputs in local served directory | CLI path isolation including parent symlink resolution; file permissions and exclusive creation. |
| Unsupported promises in docs/comments | Current architecture and README describe working behavior and explicit unavailable features. |

## Still unavailable

- **Phase 0 / original #1–3, #7, #14–15, #26–27:** no physical hardware was available
  in this session. No ordinary-app AVF deployment path or protected Waydroid
  environment has been demonstrated. Developer/root-only access cannot satisfy
  consumer distribution. GPU software rendering is a possible performance
  experiment, not proof of protected deployment.
- **Linux VM boot:** the lazy-reader/range interface works, but no bootable Linux
  guest was booted. Test with a pinned synthetic image and compatible VM tooling;
  use a disposable overlay and disabled guest networking. Record fetched bytes,
  startup latency, memory and cache behavior.
- **Recovery and migration (#6, #16–18, #28):** select a reviewed protocol and
  maintained library; specify compromise assumptions, enrollment, attempt-limit
  abuse, kit behavior, signing-authority replacement and device wrapping. Do not
  design a new OPAQUE/share protocol merely to fill the stubs.
- **Writable sessions (#8–9, #11, #30):** define durable overlays, guest flush/FUA,
  server fencing, safe conflicts and application consistency before enabling
  remote writes. Current parent-CAS works for local publishers only.
- **Sessions and endpoint protection (#19–23):** attested channel binding, protected
  runtime lifetime, session capabilities and human input/display trust require
  separate work. Offline erasure and universally safe hostile-host interaction
  cannot be supplied by this storage implementation.
- **GC and economics (#10, #33):** no GC or production quotas. Orphans and history
  accumulate; measure actual storage, egress, churn and service costs before
  choosing retention or pricing. Do not delete snapshots referenced by readers.
- **Privacy/network limits (#5, #12–13):** network availability and metadata leakage
  remain. ORAM, traffic-analysis resistance and anonymity are not implemented.
- **Product deployment (#4, #23–29):** measured phone performance, signed release
  pipeline, visible foreground operation, lender stop control, managed-device
  policy and realistic emergency-task usability remain untested.
- **Legal/business (#31–35):** actual distribution/app-policy review, launch-market
  legal review, data inventory/residency/deletion and validated economics remain.
  A code change cannot certify these gates.

The optional range server exposes plaintext to the trusted local development
host. It must not be mistaken for protection against that host. Borrowed-phone,
Essentials Vault and duress modes have no functioning UI or release path yet.

## Verification performed

Tests cover authenticated round trips; altered block contexts, keys, IDs, salts,
nonces and ciphertext; changed manifest fields and signer; geometry and bounds;
exact-checkpoint verification and reopening; local/HTTP cross-block and tail
reads; bounded byte ranges; ciphertext corruption; missing-block publication;
competing writers; stale parents; unchanged-object reuse; short source input;
identity file permissions and symlink isolation; oversized HTTP metadata; and
read-only endpoint behavior. CLI smoke checks exercise separate owner/reader
files and snapshot publication followed by a verified cross-block read.

Race tests, vet and build are recorded in the task report after execution. This
is development verification, not a cryptographic audit, power-loss/fsync fault
campaign, Android exploit test, or a claim that every phase gate has passed.
