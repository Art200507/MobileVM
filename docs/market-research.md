# Market research: is there a real product here, and for whom

Research date: 2026-10-27. Status: findings for discussion, not validated
market sizing — figures below are secondary-source reporting, re-verify
before relying on them for a business decision. This looks at who already
solves adjacent problems, where the actual gap is, and what pricing/
distribution patterns comparable products use.

## The pain point is real and large, and nobody actually solves it today

Asurion reports roughly 7.3 million phones lost or stolen in the US in 2024
— about 11,000 a day — and dominates US device protection with ~70% market
share and ~$10.6B annual revenue; the broader US mobile phone insurance
market was ~$11.1B in 2025, projected to ~$18.8B by 2031. [Asurion
data][asurion-data] [market size][market-size]

But look at what that $11B+ industry's actual answer to "I lost my phone"
is: lock/erase the lost device, file a claim, **wait 1–3 business days** for
a replacement, then restore a backup **onto the new physical device you now
own**. [Asurion replacement process][asurion-replace] There is no "use
someone else's phone in the meantime" option anywhere in this category. The
entire industry's time-to-recovery is measured in days, and its answer
requires you to end up owning new hardware — it does not, and structurally
cannot, offer "your digital life back in the next ten minutes on a phone
that isn't yours." That's the actual gap, and it's a gap in the single
largest, best-funded, most mainstream adjacent market that exists.

## Competitive landscape, mapped by what problem each category actually solves

| Category | Examples | Solves | Doesn't solve | Pricing |
|---|---|---|---|---|
| Device insurance/protection | Asurion, Assurant, AppleCare, carrier plans | Get a replacement device, repair, fraud/malware scanning | Access to your data *before* the replacement arrives | Bundled into monthly phone bill, typically $7–17/mo |
| Cloud backup/restore | Google Backup, iCloud, Samsung Smart Switch | Don't lose data permanently | Access on a device you don't own; most keep only the *latest* backup (no history); ecosystem-locked (Smart Switch is Samsung-only, iCloud→Android is lossy) | Free tier + $16–25/yr for extra storage |
| Password manager emergency access | 1Password Emergency Kit, Bitwarden Emergency Access | Let a trusted person into *your accounts* if you're unavailable | Not a phone environment at all — just credential handoff, with a multi-day waiting period by design | Bundled into $20–72/yr plans |
| Digital legacy/estate planning | Everplans, GoodTrust, Eternal Vault, Clocr | What happens to your digital accounts/assets when you die or are incapacitated | Wrong trigger entirely — these activate on death/incapacity, not "I lost my phone today"; no live device/persona concept | $20–60/yr standalone |
| Enterprise BYOD/MDM | Samsung Knox, Android Work Profile, Android Enterprise | Separate work data from personal on one device, IT-managed | Personal zero-knowledge trust model (IT controls the container, not a consumer); not triggered by device loss, built for day-to-day policy enforcement | B2B licensing, not consumer pricing; market itself is ~$121B in 2026 → ~$211B by 2030 |

[1Password/Bitwarden pricing][pw-pricing] [digital legacy
comparison][legacy-comparison] [BYOD market size][byod-market] [backup
limitations][backup-limits]

## The actual gap

Nobody combines all three of: (1) zero-knowledge personal data, (2) genuine
access on a device you don't own, within minutes, (3) triggered by "I lost
my phone right now" rather than "I died" or "my employer requires this."
Each adjacent category solves exactly one axis. That's a real, specific,
currently-empty position — not a reason to assume success, but a reason the
idea isn't redundant with something already on the market.

## The cautionary tale: don't repeat Blackphone's mistake

Silent Circle's Blackphone is the direct cautionary precedent for
"security/privacy-maximalist mobile product." It sold ~6,000 units against
a forecast of 250,000, burned through distributor deals, and the company
had to lay off 15% of staff and retreat to software-only. The causes,
per the court filings that became public: overestimated demand for a
privacy pitch narrowly aimed at "privacy enthusiasts," hardware that was
expensive and underpowered relative to mainstream phones, and a closed
ecosystem (PrivatOS) that only worked between other Blackphone owners.
[TechCrunch][blackphone-flop] [Forbes][blackphone-forbes]

Three direct lessons for this project:
1. **Never ship hardware.** ShadowDevice's own architecture already avoids
   this (it runs as software on phones people already own), but it's worth
   stating as a hard constraint, not an accident of the current design.
2. **The pitch can't be "privacy for its own sake."** Blackphone's addressable
   market was "people who care about privacy ideologically" — too small.
   ShadowDevice's addressable trigger is "I lost my phone" — something that
   happens to 11,000 people a day regardless of their views on privacy. Lead
   with the mainstream trigger, not the cryptography.
3. **Don't require a closed ecosystem.** Blackphone only worked
   phone-to-phone with itself. ShadowDevice's "any borrowed Android phone, or
   Essentials Vault on literally anything" design is already the opposite of
   this mistake — keep it that way; resist any pressure toward a
   proprietary-hardware or same-app-required special case.

## Pricing anchor

Comparable standalone subscription security/legacy products cluster in two
bands: emergency-access-as-a-bundled-feature (effectively $0 marginal cost,
riding inside a $20–72/yr password manager or VPN plan) and
standalone-digital-legacy-app pricing at $20–60/yr (GoodTrust, Eternal
Vault, Clocr). [pricing sources above] A standalone ShadowDevice consumer
subscription landing in roughly that $30–70/yr range would be
competitively positioned against the nearest comparable categories — not a
precise number, a band to sanity-check an actual pricing model against.

## Distribution: the more interesting question is who the customer is

Direct-to-consumer acquisition for a net-new security category is hard and
slow — Blackphone's failure mode. The more interesting signal from this
research: **Asurion already owns the exact customer relationship and
moment** (7.3M claims/year, "my phone is lost") that ShadowDevice's value
proposition targets, and their current offering has a multi-day gap
ShadowDevice directly fills. A licensing/bundling conversation with a
device-protection incumbent (or a carrier, who bundles these plans into
monthly bills) is a plausible B2B2C channel that sidesteps cold-start
consumer acquisition entirely — sell into a relationship that already
exists at the exact moment of need, rather than convince someone to
proactively install a security app before they ever need it (the adoption
problem flagged repeatedly in `design-review.md`).

A second, larger but differently-shaped channel: the BYOD/enterprise
market ($121B in 2026, projected $211B by 2030) already pays for
device-separation technology (Knox, Work Profile) at real budget-holder
scale. The per-category-disk architecture in `product-expansion.md` §1
maps naturally onto an enterprise pitch — "IT-managed emergency access to a
*work* persona on a lost or borrowed device" — though note this is a
different trust model (employer-controlled, not personal zero-knowledge)
and would need to be a clearly separate product line, not a repositioning
of the consumer pitch.

Neither of these is a recommendation to pursue now — they're both research
findings suggesting the consumer-direct path assumed so far is probably the
*hardest* available path, not the only one, and worth weighing against
channels that already own the customer relationship this product needs.

## Sources

[asurion-data]: https://www.asurion.com/connect/tech-tips/what-to-do-when-your-phone-is-lost-or-stolen/
[market-size]: https://www.kenresearch.com/industry-reports/usa-mobile-phone-insurance-market
[asurion-replace]: https://www.asurion.com/connect/tech-tips/how-asurion-phone-replacement-works/
[pw-pricing]: https://alternativeto.net/news/2026/2/1password-to-raise-subscription-prices-for-individual-and-family-plans-by-up-to-33-
[legacy-comparison]: https://www.getmemorial.com/blog/best-digital-legacy-services
[byod-market]: https://www.einpresswire.com/article/949269715/the-bring-your-own-device-byod-and-enterprise-mobility-market-is-projected-to-reach-211-1-billion-by-2030
[backup-limits]: https://www.androidauthority.com/restore-android-phone-from-backup-3229485/
[blackphone-flop]: https://techcrunch.com/2016/07/07/silent-circles-blackphone-revealed-as-a-sales-flop/
[blackphone-forbes]: https://www.forbes.com/sites/thomasbrewster/2016/07/06/silent-circle-blackphone-losses-layoffs-geekphone-lawsuit/
