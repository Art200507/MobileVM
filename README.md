# ShadowDevice

Your phone, streamed into a protected VM on anyone else's. See
[`docs/architecture.md`](docs/architecture.md) for the full design (condensed from
`ShadowDevice Security Architecture.pdf`): threat model, key hierarchy, block
encryption, and the phased build plan.

## Status

Scaffolding only — no phase is implemented yet. Every package in `internal/` and
every command in `cmd/` is a stub whose doc comment states which build-plan phase
owns it and what it's responsible for.

## Layout

```
cmd/
  blockstore-server/   cloud: stores encrypted blocks + signed manifests   (Phase 1)
  sync-agent/           owner phone: snapshots, encrypts, uploads          (Phase 1/5)
  guest-client/          borrowed phone: unlocks, streams, decrypts         (Phase 1-3)
  recovery-service/      cloud: OPAQUE + guess-limited key recovery         (Phase 2)
  lease-server/           cloud: single-active session lease                (Phase 5)
internal/
  protocol/   shared wire/domain types (Block, Manifest, Session)
  keys/       key hierarchy: MK -> block key / block-ID key / session cache key
  block/      fixed-4MB-block AEAD sealing + keyed block IDs
  manifest/   Merkle manifest build, sign, verify
  recovery/   OPAQUE unlock flow, recovery kit
  session/    attestation, session lifecycle, single-active lease
docs/
  architecture.md   condensed design reference
```

## Build

```
go build ./...
```

## Next step

Per the build plan, Phase 1 (`internal/block`, `internal/manifest`,
`cmd/blockstore-server`, `cmd/sync-agent`) is the first piece that's pure software
and testable on this machine without a physical device. Phase 0 (Waydroid-in-AVF
feasibility) needs a real Pixel and is tracked separately.
