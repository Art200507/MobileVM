# ShadowDevice

Development prototype for authenticated, encrypted disk snapshots and on-demand
reads. It does not yet recover a phone persona or run safely on a hostile phone.

The [current architecture](docs/architecture.md) describes implemented behavior.
The [design review](docs/design-review.md) assesses all 35 original concerns.
The [phase plan](docs/phase-1-plan.md) and [remaining work](docs/remaining-work.md)
distinguish software progress from platform and product gates.

## Working features

- Disk-scoped keys, per-object AES-256-GCM encryption and keyed object IDs.
- Signed manifests covering disk geometry, block generations and parent history.
- Immutable filesystem storage with durable publication and expected-parent CAS.
- Incremental uploads that reuse authenticated unchanged blocks.
- On-demand local/HTTP reads with one-block plaintext cache and pinned checkpoints.
- Separate owner/reader identities; read-only loopback ciphertext/range servers.

Requires the Go version in go.mod. Uses standard-library dependencies only.
Filesystem locking currently targets macOS/Linux. Tests use synthetic data.

```sh
go test -race ./...
go vet ./...
go build ./...
```

## Local demo

Use a frozen, synthetic raw image. Keep identity files outside the ciphertext
store. These identities contain plaintext secrets protected only by local file
permissions; use a trusted development machine.

```sh
mkdir -p /tmp/shadowdevice-demo
cd /path/to/shadowdevice

# Create owner.json and owner.json.reader, both with permissions 0600.
go run ./cmd/sync-agent -init \
  -identity /tmp/shadowdevice-demo/owner.json \
  -store /tmp/shadowdevice-demo/ciphertext

# The source must already be offline/frozen. The command does not snapshot a VM.
go run ./cmd/sync-agent \
  -identity /tmp/shadowdevice-demo/owner.json \
  -store /tmp/shadowdevice-demo/ciphertext \
  -image /path/to/synthetic-disk.raw -parent none
```

Record the `Manifest` digest printed by the writer. Transfer that checkpoint via a
trusted channel. For a subsequent upload, supply the last trusted digest as
`-parent DIGEST`; a stale parent is rejected. Successful uploads print new stats
and a new checkpoint. A failure after rename/fsync can have an uncertain commit
outcome; inspect the head and verify its signed manifest before retrying.

```sh
go run ./cmd/guest-client \
  -identity /tmp/shadowdevice-demo/owner.json.reader \
  -store /tmp/shadowdevice-demo/ciphertext \
  -manifest DIGEST -offset 0 -length 4096 \
  -out /tmp/shadowdevice-demo/first-page.bin
```

Output files must be new; existing files are not overwritten. Stats go to stderr.
To serve ciphertext, run the following in another terminal and replace the
reader's `-store ...` argument with `-url http://127.0.0.1:8080`:

```sh
go run ./cmd/blockstore-server \
  -store /tmp/shadowdevice-demo/ciphertext -listen 127.0.0.1:8080
```

For local VM tooling, replace the reader's `-out ...` with
`-listen 127.0.0.1:8081`. This serves **plaintext** at `/disk.raw`, supports HEAD and
bounded single byte ranges, and requires a trusted local host. For example,
`curl -H 'Range: bytes=0-4095' http://127.0.0.1:8081/disk.raw` reads one range.
A VM boot gate remains unverified; this endpoint is not an Android pVM integration.

## Limits

No PIN recovery, protected Android sessions, writable remote disks, Essentials
Vault, enforced leases/timeouts, or automatic latest-state discovery. Their
placeholder services fail closed. No anonymity, forensic erasure, instantaneous
remote kill or coercion-resistance guarantee. See [SECURITY.md](SECURITY.md).
