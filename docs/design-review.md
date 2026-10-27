# ShadowDevice design review

Research date: 2026-10-21. Status: recommendations for discussion, not an approved
production security specification. No hardware experiments were performed.

This review supersedes conflicting feasibility and security claims in the v5
PDF. The PDF remains the historical proposal. Source links below were checked
on the research date; moving AOSP branches describe upstream code, not a promise
about shipping firmware. Existing Go types remain unimplemented proposals.

## Recommendation

Continue the encrypted storage research, but stop presenting an arbitrary borrowed
phone as a trustworthy recovery endpoint. Keep three independently gated tracks:

1. Encrypted, authenticated disk streaming on a trusted development client.
2. Protected execution on explicitly tested Android builds, with a viable deployment path.
3. Emergency access with a clearly stated input/display trust boundary.

Track 1 can proceed after the protocol decisions in [phase-1-plan.md](phase-1-plan.md).
Tracks 2 and 3 are not established by a successful lazy-boot demonstration.
An Essentials Vault is a possible narrower product, not an automatic solution to
hostile input, malicious browser code, or unsafe recovery credentials.

## Evidence that changes the proposal

### Ordinary app installation does not grant AVF access

AOSP's API guide identifies AVF APIs as restricted System APIs. The current
manifest grants VM management to signature/preinstalled/development callers and
custom VM access to signature/development callers. ADB development grants do not
establish consumer deployability. [API guide][avf-api], [permissions][permissions].

Conclusion: the proposed independently distributed app has a platform-access
blocker under this documented interface. A preinstalled/OEM route or a different,
documented interface could change that assessment. Do not infer permission from
the presence of Terminal, a chipset name, or an AVF feature flag.

### GPU acceleration is not an absolute Waydroid requirement

Waydroid documents a SwiftShader software-rendering configuration for VMs.
Its architecture uses Linux containers, not an additional hardware VM.
[VM instructions][waydroid-vm], [project description][waydroid].

Correction to the earlier assessment: lack of protected GPU access need not make
all Android application execution impossible. It may make it too slow or costly;
that requires measurement. The relevant stack is Android apps inside a Waydroid
container inside a Linux VM managed by AVF, not three nested hardware hypervisors.
Guest kernel features, boot authorization, memory, and graphics sharing remain
separate constraints.

### Terminal graphics do not establish protected Android execution

The Android Linux VM presentation describes the Debian VM as non-protected.
The AVF security documentation restricts protected boot images to approved
signers and says applications cannot bundle their own kernels.
[Linux VM presentation][terminal], [AVF security][avf-security].

Conclusion: neither Terminal nor a custom-kernel development demo demonstrates a
deployable protected Waydroid environment. No tested Pixel/Samsung compatibility
list exists for ShadowDevice. Protected GPU support remains unverified here;
this review does not claim universal hardware impossibility.

### Attestation needs to identify the execution endpoint

The upstream VM-attestation guide currently documents Microdroid support, a fresh
challenge, payload measurements, and a private key confined to the pVM.
[VM remote attestation][vm-attestation].

Design implication: checking an app token is insufficient. Recovery must verify
the intended payload and bind secret delivery to an authenticated channel whose
private key stays in that protected instance. Otherwise the host could relay a
valid attestation while requesting delivery elsewhere. A protected key broker
cannot preserve full-persona confidentiality if it subsequently releases disk
keys or bulk plaintext to an unprotected Debian VM.

### OPAQUE does not make an ordinary server's PIN limit breach-resistant

RFC 9807 explicitly permits offline dictionary attacks after server compromise;
key stretching raises cost but does not eliminate the attack. Signal's SVR3 uses
multiple hardware-protected backends and an OPRF protocol, not merely replicated
ordinary share databases. [RFC 9807, section 10.11][opaque], [SVR implementation][svr].

Conclusion: an ordinary-server prototype must trust its recovery operator and
server secrets. A six-digit PIN plus that prototype cannot support the PDF's
blanket breached-cloud guarantee. Specify exactly which server records and
secrets an attacker obtains. Do not invent an OPAQUE/Argon2/share composition;
choose a reviewed protocol and implementation before Phase 2.

### Hardware key protection and recoverability are different properties

Android Keystore protects its keys from extraction and lets applications request
cryptographic operations using them. [Keystore documentation][keystore].

The proposal needs portable encrypted data-key material if MK is to be recovered
on another phone. Recommended model: a random portable data root, wrapped locally
by a non-exportable device key and separately by a recovery mechanism. State where
the unwrapped root exists. Keep signing authority separate: recovering a data key
must not silently grant unrestricted permission to replace account history.
Hardware support for the chosen signing algorithm must be tested, not assumed.

## Fundamental limits and threat boundaries

These are deductions from the proposed trust model, not hardware test results.

- A host-controlled keyboard/display can record credentials, spoof prompts,
  observe displayed data, and inject actions. pVM memory isolation does not
  authenticate the interface to the human. Scrambling a keypad cannot defeat a
  host that sees both its layout and touch coordinates.
- One-time account-code rotation does not stop real-time phishing or a race to
  use captured credentials. A high-entropy recovery kit typed into a hostile
  endpoint is also exposed. An external trusted factor must authenticate the
  transaction/session; simply approving a generic login is insufficient.
- A scoped key limits underlying data exposure. A UI showing only essentials
  while retaining MK does not. Use a separately keyed essentials dataset if that
  isolation is desired; decide whether backup codes belong in its default scope.
- Revocation can deny future server operations. It cannot erase plaintext or keys
  already copied, stop an offline lost phone, or undo external app actions.
- Persistent access to MK lets a recipient derive the existing hierarchy. A
  timeout on the cache key does not revoke MK. Enforced expiry requires trusted
  execution and a defensible clock/lifecycle design, or narrower server-mediated
  capabilities with their own availability tradeoff.
- A signed old snapshot is still authentic. A new client with no trusted history
  cannot identify the newest state using signatures and version numbers alone.
- Host compromise is distinct from compromise of the persona OS or an app inside
  it. Once the guest itself is compromised, pKVM does not hide its data from it.

For web mode, W3C identifies script injection as a key/data-exfiltration threat
and does not require key zeroization. Browser, origin code, extensions, and host
must be within the trusted endpoint boundary. [WebCrypto security][webcrypto].
Disabling caches and using CSP reduce exposure; they do not establish forensic
erasure or safety on a malicious endpoint. A server able to replace the delivered
web application may attack decryption even if the block store contains ciphertext.

## Disposition of all 35 reported concerns

“Blocker” means blocker to the original promise, not to every narrower project.
“Before P1” means a protocol or scope choice must precede implementation.

| # | Assessment | Required response |
|---|---|---|
| 1 | Unproven platform constraint | Test protected execution and rendering separately; try software rendering; inspect every shared buffer. |
| 2 | Documented deployment blocker | Establish an authorized consumer launch path; record ADB/root/OEM dependencies. |
| 3 | Deferred capability | Omit sensors from the initial scope; later evaluate permissions, host spoofing, privacy, and guest drivers. |
| 4 | Measurable tradeoff; premise corrected | Waydroid is a container. Measure actual CPU, memory, startup, thermal and energy costs. |
| 5 | Availability limit | Define offline behavior, data caps, latency targets, and failure messages; connectivity cannot be guaranteed. |
| 6 | Before P1 key-model decision | Separate portable data root from device wrapping key; design migration and signing-key replacement. |
| 7 | Ongoing platform risk | Qualify by exact model/build/configuration and retest updates; chipset labels are insufficient. |
| 8 | Before P1 durability decision | Start read-only; later implement durable overlays, flush semantics, and atomic commits. |
| 9 | Before P1 concurrency decision | Parent hashes, compare-and-swap and fencing; preserve conflicting branches rather than silently overwrite. |
| 10 | Manageable storage economics | Reachability-based GC with retention, in-flight pins, quotas, and lifecycle accounting. |
| 11 | Before P1 ordering decision | Use logical versions and parent hashes; separate monotonic local durations from server lease authority. |
| 12 | Accepted metadata leakage initially | Declare block-access visibility; fixed sizes and keyed names are not ORAM. |
| 13 | Accepted metadata leakage initially | Minimize logs and retention; proxies shift trust and do not guarantee anonymity. |
| 14 | Explicit root-of-trust dependency | Specify accepted roots, payloads, versions, revocations, and delivery-channel binding. |
| 15 | Compatibility/security tradeoff | Fail closed for protected mode; do not quietly downgrade its guarantees. Custom ROM status alone is not a complete security verdict. |
| 16 | Before recovery/lease implementation | Discovery code alone must not revoke sessions or destroy shares; separately address failed-login exhaustion. |
| 17 | Availability dependency | Recovery kit must bypass recovery service, but still requires accessible ciphertext and a trusted client. Test outages. |
| 18 | Threat-model/legal dependency | Crypto cannot prevent denial of service or metadata collection; ordinary recovery server remains trusted. |
| 19 | Unsupported protection claim | Defer duress mode; no promise that a decoy defeats coercion or hides its own existence. |
| 20 | Endpoint trust limit | Distinguish host from guest compromise; displayed content and typed secrets can escape. |
| 21 | Weaker endpoint mode | Separate web guarantees; no master-persona key in an essentials-only browser client. |
| 22 | Root-of-trust limit | Record owner compromise as outside recovery protection; wrapping does not cure authorized malicious key use. |
| 23 | Distribution trust dependency | Signed releases, controlled updates, provenance, minimal permissions, and clear lender consent. |
| 24 | Product abuse risk | Visible foreground sessions, lender stop control, no stealth install/persistence or host-data collection. |
| 25 | Deployment/consent constraint | Respect managed-device policy; no bypass instructions; consent from lender and persona owner. |
| 26 | Adoption blocker for original flow | Installation authenticity and restricted AVF privileges both need answers; preinstallation/OEM route changes product scope. |
| 27 | Unsupported coverage claim | Replace model promises with tested capabilities; browser access retains endpoint risks. |
| 28 | Adoption hypothesis | Test a short setup and successful restore rehearsal; measure completion rather than assume backups exist. |
| 29 | Adoption hypothesis | Test short emergency tasks; define success without assuming a stranger lends a phone for 30 minutes. |
| 30 | Product/concurrency tradeoff | Separate readers from writers; single-writer storage does not require a single viewing session. |
| 31 | Distribution review gate | Review actual store/OEM/app terms for chosen implementation; no blanket WhatsApp or Play Integrity guarantees. |
| 32 | Claim accuracy gate | Publish per-mode guarantees; remove “nothing left,” instantaneous kill, and coercion-resistance absolutes. |
| 33 | Unvalidated economics | Measure retained bytes, churn, egress, operations, enclave cost and support per account; do not quote invented prices. |
| 34 | Jurisdiction review gate | Determine launch markets and actual cryptographic distribution before making compliance conclusions. |
| 35 | Privacy/legal review gate | Inventory account, IP, access, billing and encrypted-content data; define retention, deletion and processor regions. |

Encryption must not be treated as automatic anonymization. EDPB distinguishes
pseudonymisation from anonymisation; jurisdiction-specific obligations still need
review against the actual service and data flows. [EDPB overview][edpb]. This
review makes no determination about individual laws, export classifications,
store acceptance, or third-party application terms.

## Additional concerns discovered

1. **Signer bootstrap:** fetching the manifest public key from the same malicious
   storage server does not authenticate it. Provision a trust anchor independently.
2. **Unsigned semantics:** current signatures cover only account/version/root.
   The exact disk length, disk identity, algorithms, key epoch, parent history,
   and any operationally relevant fields must also be authenticated.
3. **Verifier expectations:** the current block Open API receives only the block.
   It needs independently authenticated expected position and generation; comparing
   a supplied field to itself does not prevent substitution.
4. **Repeated recovery leaks historical data:** someone retaining MK and old
   ciphertext retains decryption capability. Revoking a session is not key rotation;
   rotating wrappers alone does not revoke the underlying data key.
5. **Read-only does not mean side-effect-free:** booted apps can send messages,
   refresh tokens, or contact external services even with a disposable disk overlay.
   First demos should disable guest networking or use a controlled test workload.
6. **Application snapshots:** a filesystem-consistent image need not be an
   application-consistent backup. Pause/quiesce the source or use supported
   snapshot interfaces and explicitly document the consistency level.
7. **Availability attacks:** a valid signed disk can still be withheld. Cache,
   replica, or kit availability must not be confused with integrity verification.
8. **Secret lifecycle:** Go values, garbage collection, crash reports, swap and
   diagnostic logs complicate zeroization. A zeroed array alone proves no global
   erasure property. Secure runtime behavior requires separate verification.
9. **Bootstrapping encryption:** the fetch/decrypt adapter must run inside the
   intended trust boundary. Host-side decryption for a convenience demo is not a
   protected-storage design.

## Decisions and experiments in priority order

1. Adopt the limited Phase 1 scope in the companion plan as a proposed research
   baseline. Make the recovery trust assumptions explicit before selecting crypto.
2. Establish an ordinary-app deployment path before spending time optimizing
   protected Waydroid graphics. A root-only result cannot pass this gate.
3. On available hardware, record exact firmware, privileges, protected status,
   kernel provenance, attestation support, rendering path, and measured workload.
   Use only synthetic data. Run software-rendered and accelerated cases separately.
4. Test that a secret is delivered to the measured pVM endpoint and remains there.
   Reject wrong payloads, stale challenges, debug configurations and wrong keys.
5. Test usability using a specific emergency task and explicit lender consent.
   Do not market safe hostile-phone recovery until the input/output problem has a
   stated, demonstrated solution or the promise has been narrowed.

No actual phone, malicious-host, network-performance, recovery-service, or
policy-compliance tests were performed in this review.

## Sources

[avf-api]: https://android.googlesource.com/platform/packages/modules/Virtualization/+/refs/heads/main/libs/framework-virtualization/README.md
[permissions]: https://android.googlesource.com/platform/packages/modules/Virtualization/+/refs/heads/main/android/android.system.virtualmachine.res/AndroidManifest.xml
[avf-security]: https://source.android.com/docs/core/virtualization/security
[terminal]: https://lpc.events/event/19/contributions/2123/attachments/1730/3786/LPC%202025%20-%20A%20Linux%20VM%20on%20Android%20via%20AVF%20%28Android%20MC%29.pdf
[waydroid-vm]: https://docs.waydro.id/faq/get-waydroid-to-work-through-a-vm
[waydroid]: https://waydro.id/
[vm-attestation]: https://android.googlesource.com/platform/packages/modules/Virtualization/+/refs/heads/main/docs/vm_remote_attestation.md
[opaque]: https://www.rfc-editor.org/rfc/rfc9807.html#section-10.11
[svr]: https://github.com/signalapp/SecureValueRecovery2
[keystore]: https://developer.android.com/privacy-and-security/keystore
[webcrypto]: https://www.w3.org/TR/webcrypto/#security-considerations
[edpb]: https://www.edpb.europa.eu/topics/ai-and-technology/anonymisation-pseudonymisation_en
