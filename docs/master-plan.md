# Master plan: architecture, implementation, and what's still a decision

Status: synthesis of all research to date, 2026-10-28. This is a planning
document, not new claims of its own — every technical/market assertion
below is sourced in the doc it's drawn from. Read this as the place that
ties `architecture.md`, `design-review.md`, `phase-1-plan.md`,
`remaining-work.md`, `product-expansion.md`, `ios-platform-research.md`,
`market-research.md`, and `data-extraction-methods.md` together into one
roadmap, plus new research closing two gaps that were previously open
questions: the recovery protocol, and the Linux-boot gate.

## What this project actually is, in one paragraph

Two products sharing one crypto core. **Persona VM** (full "my apps, on a
borrowed phone" mode): works only on AVF-capable Android, gated on a
platform-access problem that isn't solved yet. **Essentials
Vault/structured extraction** (a scoped, per-category dataset — contacts,
photos, documents): works everywhere, including iOS, and is buildable
today with zero platform blockers. The honest framing, now that the
research is in: the second product is not a fallback for the first, it's
the one with no external dependency standing between here and shipping.

## Two new findings that close previously-open gaps

### Recovery: don't design a new protocol, adapt an open-source, production one

`design-review.md` flagged this as unresolved: "an ordinary server with
OPAQUE and a guess counter does not withstand server compromise... choose a
reviewed protocol." That protocol already exists, is open source, and is
running in production at tens-of-millions-of-users scale:

- **OPAQUE login layer**: `bytemare/opaque`, a Go implementation of RFC
  9807, maintained by one of the RFC's own authors, actively updated (last
  update May 2026). Matches the project's existing Go stack directly.
  [bytemare/opaque][opaque-go]
- **Guess-limit enforcement**: Signal's **SVR3** (`signalapp/SecureValueRecovery2`,
  open source) is the exact split-trust design `design-review.md` pointed at
  as the production direction — it splits the secret across three different
  confidential-computing enclaves from three different vendors on three
  different clouds (Intel SGX on Azure, AMD SEV-SNP on Google Cloud, AWS
  Nitro on AWS), and "unconditionally deletes key material after a fixed
  number of PIN guesses." [SVR3 GitHub][svr3-repo] [SVR3 paper][svr3-paper]

This changes Phase 2 from "select a protocol" to "integrate two existing,
audited open-source systems." It also changes the infrastructure cost
picture: confidential computing "moved from pilot to default... showing up
in procurement checklists" through 2026, meaning the three-cloud-enclave
requirement SVR3 has is now a commodity purchase, not custom infra work.
[Confidential Computing Summit 2026][cc-2026]

Recommended path: single-enclave MVP first (one cloud, one confidential-computing
backend, explicit documented caveat that this doesn't yet have SVR3's
split-trust guarantee), then graduate to the real three-vendor split before
calling recovery production-ready. Don't skip straight to full SVR3 if it
delays validating the rest of the product — but don't ship a single-enclave
version without the SECURITY.md caveat being explicit about exactly what
it doesn't yet guarantee.

### The Linux-boot gate (Phase 1B) has concrete, current tooling

`phase-1-plan.md` left this as "select for the development host (for
example, a Linux NBD/ublk adapter)." That's now a specific, buildable path:
**ublk** (in-kernel since Linux 6.0, userspace block devices via io_uring)
plus **nbdublk** turns an NBD server into a real `/dev/ublkbN` block device;
alternatively QEMU has a built-in NBD client (`-drive
file=nbd:host:port,if=virtio`) that needs no kernel module at all.
[ublk overview][ublk] [nbdublk][nbdublk-tool] [qemu-nbd][qemu-nbd]

Concretely: wrap `image.Reader` (already does verified on-demand decrypt)
in an NBD server instead of the current plaintext HTTP range endpoint, then
boot a QEMU VM with `-drive file=nbd:127.0.0.1:PORT`. This is the actual
next engineering step to close Phase 1B's "boots without a full download"
gate — it's a protocol-translation layer in front of code that already
exists, not new cryptography or new trust decisions.

## Platform execution: two tracks, different timelines

| | Android (AVF-capable subset) | iOS / everything else |
|---|---|---|
| Blocker | `android.permission.MANAGE_VIRTUAL_MACHINE` is a restricted System API — confirmed again this session, no third-party early-access program exists as of this research [source][avf-restricted] | None. `WKWebView` JIT + App Clips are both usable today, Apple-sanctioned |
| What ships | Full Persona VM | WASM-accelerated Essentials Vault, instant-launch via App Clip/QR |
| Status | Blocked on a platform decision outside this project's control | Buildable now |
| Action | Nothing to build until access changes — see "what's needed from you" below | Start now |

This asymmetry is the single most important planning fact: **don't
sequence the roadmap as if Android-full-VM is the main line and iOS is the
reduced version.** Build order should follow what's actually unblocked.

## Revised phased roadmap

1. **Structured record format + selective extraction** (`product-expansion.md`
   §1/§5, `data-extraction-methods.md`) — no crypto changes needed, reuses
   existing per-`DiskID` key scoping. Android: content providers gated by
   the OS's own per-category permissions (Photos/Contacts/Documents
   straightforward; SMS/Call Log require becoming the default handler,
   treat as a separate scoped decision, not default scope). iOS: equivalent
   frameworks, except Messages, which has no API on any iOS version,
   period. **Buildable now, no external dependency.**
2. **Phase 1B Linux boot gate** — ublk/nbdublk or qemu-nbd in front of the
   existing `image.Reader`. **Buildable now.**
3. **iOS Essentials Vault** — `WKWebView`-hosted, WASM-accelerated,
   launchable via App Clip for the no-install borrowed-phone case.
   **Buildable now.**
4. **Recovery (Phase 2)** — `bytemare/opaque` + single-enclave MVP, then
   graduate to full three-vendor SVR3 split. **Buildable now** (the
   research gap is closed); real engineering effort, not a research
   blocker.
5. **Compression with size-tiering** (`product-expansion.md` §2) — the
   `internal/compress` building block exists; wiring it in requires the
   tiered-size wire-format decision. **Buildable now**, deliberately
   sequenced after the above since it's a pure optimization.
6. **Protected Android execution (Phase 3, full VM)** — genuinely blocked.
   No action until the platform-access picture changes (see below).
7. **Owner sync, single-active lease, duress mode (Phase 5)** — depends on
   recovery (step 4) and, for the full-VM case, step 6.

## Business and compliance findings

- **Funding is realistic, with a real comp**: cybersecurity/privacy
  startups raised $4.4B in Q2 2026 seed-through-growth (down ~30%
  quarter-over-quarter and year-over-year — a real caution, not a boom
  market right now), average seed round ~$4.8M ($3–7M typical range). The
  closest direct comp: **Cloaked, a consumer privacy/security platform,
  raised a $375M Series B** — strong evidence investors will fund this
  category seriously when the pitch lands. [Crunchbase][funding-2026]
  [Cloaked round][cloaked]
- **SOC 2 + cyber liability insurance are real prerequisites**, not later
  nice-to-haves, specifically because `market-research.md`'s recommended
  channel (licensing/bundling with a device-protection incumbent or
  carrier) runs through enterprise procurement: 67% of enterprise
  procurement teams require proof of cyber insurance, typically $1–5M
  coverage, and SOC 2 Type II compliant companies pay $3,000–$12,000/year
  for it. Underwriters specifically check for AES-256 at rest and TLS 1.2+
  in transit — **already satisfied by the existing architecture** — and
  MFA, which recovery (step 4) effectively is. [SOC 2/insurance
  research][soc2-insurance] This means: if the B2B2C channel is pursued,
  SOC 2 readiness should be a parallel workstream from early on, not
  something addressed after product-market fit, since it's a hard gate for
  the specific channel that looks most promising.

## What's needed from you

Research can't resolve these — they're either business decisions or things
outside this project's control:

1. **Build priority.** The roadmap above is ordered by what's technically
   unblocked (structured extraction → Linux boot → iOS Essentials →
   recovery → compression), not by what you consider most important. Confirm
   or reorder.
2. **Business direction.** Bootstrap, raise seed funding, pursue a
   licensing/bundling conversation with a device-protection incumbent or
   carrier, or stay open-source/non-commercial? This changes whether SOC 2
   /insurance prep starts now or later, and whether legal-entity formation
   is a near-term task.
3. **Cloud budget.** R2 (or equivalent) storage is cheap; a real SVR3
   three-vendor enclave deployment (SGX+SEV-SNP+Nitro across three clouds)
   is a recurring cost once past the single-enclave MVP. Need a signal on
   what's budgeted before step 4 goes past the MVP stage.
4. **Any existing Google relationship.** No public third-party AVF
   early-access program exists as of this research. If you have any
   existing contact at Google (Android partnerships, a developer relations
   contact, anything), that's the only lever that could change the
   Android-full-VM timeline — I have no path to pursue this from here.
5. **Whether to keep going on code now**, or pause for you to review this
   plan first. Given the pattern so far, my default is to start on item 1
   of the roadmap (structured record format) unless you say otherwise.

## Sources

[opaque-go]: https://github.com/bytemare/opaque
[svr3-repo]: https://github.com/signalapp/SecureValueRecovery2
[svr3-paper]: https://eprint.iacr.org/2024/887.pdf
[cc-2026]: https://confidentialcomputing.io/2026/07/02/confidential-computing-summit-2026-six-signals-from-the-year-the-foundation-got-real/
[ublk]: https://hackaday.com/2026/01/20/block-devices-in-user-space/
[nbdublk-tool]: https://libguestfs.org/nbdublk.1.html
[qemu-nbd]: https://www.qemu.org/docs/master/tools/qemu-nbd.html
[avf-restricted]: https://android.googlesource.com/platform/packages/modules/Virtualization/+/refs/tags/aml_net_351410000/libs/framework-virtualization/README.md
[funding-2026]: https://news.crunchbase.com/cybersecurity/solid-startup-venture-funding-growth-h1-2026/
[cloaked]: https://news.crunchbase.com/venture/biggest-funding-rounds-security-ai-cloaked-frore/
[soc2-insurance]: https://insura.ai/articles/soc2-cyber-insurance-saas-guide
