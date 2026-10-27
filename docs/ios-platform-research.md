# iOS platform research: what's actually possible

Research date: 2026-10-27. Status: findings for discussion, not a build plan. Web
sources are secondary reporting, not Apple engineering documentation; treat
specifics (exact version numbers, size limits) as directional and re-verify
against `developer.apple.com` before depending on them in an implementation.
This supersedes the iOS assessment in the v5 PDF and architecture-v5.md, which
predates this research and was more pessimistic in one respect (JIT) and not
pessimistic enough in another (cloud streaming).

## The headline conclusion

Full VM-based persona execution is not coming to iOS through any legitimate
path available today. That part of the original PDF's pessimism was correct,
and if anything the cloud-streaming alternative is *more* blocked by policy
than it is by technology. But two narrower things are more possible than the
original assessment assumed: (1) a WASM-compiled emulator inside a WKWebView
gets real JIT speed, not interpreter speed, and (2) App Clips are a
legitimate, zero-install way to get *something* running on a stranger's
iPhone in seconds — which attacks a different problem (adoption) than the one
the PDF was focused on (execution).

## 1. No third-party VM execution, and it isn't close

`Hypervisor.framework` existed for third-party apps only briefly: open from
iOS 14.5 through 16.3 (this is how UTM ran accelerated VMs on M1/M2 iPads),
then Apple pulled it in 16.4. Static reverse-engineering of iOS 26's
kernelcache (April 2026) shows the hardware hypervisor capability and kernel
code are still physically present, gated by a device-tree property — but
that is not a public API, and Apple has already demonstrated once that it
will revoke this exact capability from third parties. Do not plan a product
around it reappearing. [Husk issue][husk] [worthdoingbadly][wdb]
[ios26-hypervisor-research][hv-research]

There is no narrower "just the memory-isolation part" or "just for Linux
guests" exception either — this is an all-or-nothing platform capability, and
it's currently nothing for third-party apps.

## 2. Cloud-streaming the persona is blocked by policy, not just technology

This is the more important correction to make to the original plan. The
instinct — "if we can't run the VM on the iPhone, stream it from our cloud
like GeForce Now" — runs into an explicit, current App Store rule:

> Thin clients for cloud-based apps are not appropriate for the App Store.

Apple's remote-mirroring guideline only permits connecting to a **host device
the user owns**, on a **local/LAN connection** — i.e., "control your own PC
from your phone," not "stream a persona from our servers." [MacRumors
2018][macrumors-2018]

The one carve-out that permits true cloud streaming is guideline 4.9, and
it's narrow by design: it covers game-streaming services specifically, and
**every individual game must be submitted to the App Store as its own listing
with its own product page** — a catalog/launcher app alone doesn't satisfy
it. ShadowDevice streaming "your personal computing environment" doesn't fit
this shape at all, and there's no indication Apple would extend it by
analogy. [iClarified][iclarified] [MacRumors 2020][macrumors-2020]

Consequence: a cloud-rendered-persona-streamed-to-iPhone product cannot ship
on the primary App Store, as a matter of policy, independent of whether the
engineering works. It also still conflicts with the "cloud never sees
plaintext" rule, since the VM running your persona would have to live
somewhere that renders it to video — i.e., somewhere with your decrypted
disk. Streaming doesn't just face a policy wall, it's a different, weaker
trust model than the rest of this project.

## 3. WKWebView gets real JIT — this changes the Essentials Vault performance story

The original PDF said "Apple blocks JIT, so VMs run very slowly." That's
true for a third-party app's own process, but **not** inside a `WKWebView`
embedded in that app. `WKWebView` renders content in a separate WebContent
process that Apple itself grants the JIT entitlement to — this has been true
since iOS 14.5, and unlike `Hypervisor.framework`, Apple has not withdrawn it.
[WasmWebViewRuntime][wasmview] [HN discussion][hn-jit]

Concretely, this means: a WASM-compiled lightweight emulator or interpreter,
loaded as web content inside a `WKWebView` hosted by the ShadowDevice iOS
app, gets near-native JIT-compiled speed for that WASM code — not the
software-interpreter speeds the original PDF assumed. There's a working
proof-of-concept of exactly this pattern (compile an emulator to WASM, run it
in a hosted `WKWebView`, let WebKit's JIT compile the translated guest code)
and a dedicated open benchmark project measuring it on current hardware.
[WasmWebViewRuntime][wasmview] [wkwebview-jit-bench][jit-bench]

This doesn't revive full Android-app compatibility (that's a different, much
harder problem — running an actual Android runtime, not a generic emulator,
inside a browser sandbox). But it meaningfully upgrades what "Essentials
Vault on iOS" can be: instead of a plain decrypt-and-display web page, it
could host a real lightweight interactive environment (a terminal, a
document viewer with real rendering, conceivably a constrained Linux
userspace via an in-browser emulator like a WASM port of a RISC-V/x86
system) at genuinely usable speed. Scope and measure before promising
specific capability — "faster than assumed" is not "fast."

## 4. App Clips: a legitimate answer to a different problem

App Clips don't help with VM execution at all — they explicitly cannot
access the host phone's Photos, Contacts, Messages, or other apps' data, and
lack CloudKit/HealthKit/HomeKit/etc. But ShadowDevice's Essentials Vault
doesn't want the *borrowed* phone's data anyway — it only needs enough of a
sandbox to fetch ciphertext from the network and decrypt/display it. That's
exactly what an App Clip can do.

This directly answers one of the two problems flagged as most load-bearing
in the earlier design review: "the stranger needs the app pre-installed."
App Clips launch instantly from a QR code, NFC tag, or link — no App Store
visit, no install step, up to 100 MB for link-launched flows on iOS 17+.
[newly.app][appclip-size] A borrowed-phone recovery flow becomes: show a QR
code (printed in the recovery kit, or displayed on another device), the
stranger's iPhone scans it, an App Clip opens in seconds, hosts the
`WKWebView` Essentials experience from #3, and — because App Clips are
*meant* to be ephemeral, Apple may reclaim/expire them automatically — that
ephemerality is a feature here, not a limitation to fight. [App Clips launch
methods][appclip-launch]

This doesn't solve attestation (an App Clip proves *an* app launched, not
that a specific trusted binary with a specific key is running) or the
screen/input trust problem from the design review — those limits still
apply in full. It solves distribution friction, nothing else.

## 5. EU: alternative marketplaces are real now, but don't actually unblock the hard parts

Under the DMA, alternative iOS app marketplaces are live in the EU (and now
Japan) as of 2026 — e.g., Onside. [TechCrunch][techcrunch-dma] This matters
less than it sounds for ShadowDevice: alternative-marketplace apps still go
through Apple notarization (a lighter, but real, security check), and
nothing in the DMA changes or removes the underlying platform facts in
sections 1–2 — there's still no third-party Hypervisor.framework access, and
Apple's policy rationale against cloud-thin-clients isn't an App-Store-only
rule, it reflects how the OS and review process work. The practical win from
alternative marketplaces is narrower: distributing a *fuller-featured, more
powerful Essentials Vault app* in the EU without the main Store's review
constraints — e.g., looser guideline interpretation, no 4.9 problem for a
hypothetical future streaming mode — but it's an EU/Japan-only distribution
channel, not a technology unlock, and the roster of alternative stores has
already proven unstable (one shut down in February 2026 within months of
opening). Don't design a core dependency on any single alternative store.

## Recommendation

Treat iOS as two separate product surfaces, not one degraded version of
Android:

1. **Baseline Essentials Vault**, available everywhere (any phone/browser,
   already in the architecture) — unaffected by any of this research.
2. **iOS-specific enhanced Essentials**, built as a `WKWebView`-hosted,
   WASM-accelerated experience, reachable either from the full app or
   instantly via an **App Clip** for the no-install borrowed-phone case.
   This is buildable today, uses only Apple-sanctioned mechanisms, and
   directly targets the adoption blocker rather than chasing VM parity iOS
   structurally cannot offer.

Do not build toward cloud-streamed VM execution on iOS. It is blocked by
policy now, conflicts with the project's own zero-knowledge-cloud rule
independent of policy, and there is no credible signal Apple intends to
change course here (the opposite trend — notarization even for sideloaded EU
apps — points toward more review, not less).

## Sources

[husk]: https://github.com/Leviidev/Husk/issues/146
[wdb]: https://worthdoingbadly.com/hv/
[hv-research]: https://github.com/cloverfield11/ios26-hypervisor-research
[macrumors-2018]: https://www.macrumors.com/2018/06/04/apple-revamps-app-store-guidelines-ios-12/
[iclarified]: https://www.iclarified.com/77531/apple-updates-app-store-review-guidelines-allows-streaming-games-if-each-title-is-submitted-for-review
[macrumors-2020]: https://www.macrumors.com/2020/09/11/apple-updates-app-store-review-guidelines/
[wasmview]: https://github.com/adammharris/WasmWebViewRuntime
[jit-bench]: https://github.com/gagarinyury/wkwebview-jit-bench
[hn-jit]: https://news.ycombinator.com/item?id=40726948
[appclip-size]: https://newly.app/guides/app-clips
[appclip-launch]: https://www.bairesdev.com/blog/app-clips-vs-instant-apps/
[techcrunch-dma]: https://techcrunch.com/2026/02/22/move-over-apple-meet-the-alternative-app-stores-available-in-the-eu-and-elsewhere/
