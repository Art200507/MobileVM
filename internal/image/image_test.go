package image

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"shadowdevice/internal/block"
	"shadowdevice/internal/keys"
	"shadowdevice/internal/manifest"
	"shadowdevice/internal/protocol"
	"shadowdevice/internal/store"
)

func TestPublishLazyHTTPReuseAndConflicts(t *testing.T) {
	root := t.TempDir()
	s, e := store.Open(root)
	if e != nil {
		t.Fatal(e)
	}
	mk, _ := keys.GenerateMasterKey()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	disk := protocol.DiskID{1}
	data := bytes.Repeat([]byte{31}, 2*protocol.BlockSize+123)
	first, stats, e := Publish(s, bytes.NewReader(data), uint64(len(data)), mk, priv, disk, protocol.Digest{})
	if e != nil {
		t.Fatal(e)
	}
	if stats.UploadedBlocks != 3 {
		t.Fatal(stats)
	}
	server := httptest.NewServer(s.Handler())
	defer server.Close()
	remote, e := store.NewRemote(server.URL)
	if e != nil {
		t.Fatal(e)
	}
	r, e := NewReader(remote, mk, pub, disk, first)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	if r.Stats().FetchedBlocks != 0 {
		t.Fatal("eager block fetch")
	}
	p := make([]byte, 20)
	if _, e = r.ReadAt(p, 10); e != nil || !bytes.Equal(p, data[10:30]) {
		t.Fatal(e)
	}
	if r.Stats().FetchedBlocks != 1 {
		t.Fatal(r.Stats())
	}
	if _, e = r.ReadAt(p, 30); e != nil || r.Stats().FetchedBlocks != 1 {
		t.Fatal("cache miss", e)
	}
	crossing := make([]byte, 30)
	if _, e = r.ReadAt(crossing, protocol.BlockSize-10); e != nil || !bytes.Equal(crossing, data[protocol.BlockSize-10:protocol.BlockSize+20]) {
		t.Fatal(e)
	}
	tail := make([]byte, 200)
	n, e := r.ReadAt(tail, int64(len(data)-123))
	if n != 123 || e != io.EOF || !bytes.Equal(tail[:n], data[len(data)-123:]) {
		t.Fatal("tail", n, e)
	}
	if _, e = r.ReadAt(p, -1); e == nil {
		t.Fatal("negative offset")
	}
	second, stats, e := Publish(s, bytes.NewReader(data), uint64(len(data)), mk, priv, disk, first)
	if e != nil || stats.UploadedBlocks != 0 || stats.ReusedBlocks != 3 {
		t.Fatal("reuse", stats, e)
	}
	data[protocol.BlockSize+7] ^= 1
	third, stats, e := Publish(s, bytes.NewReader(data), uint64(len(data)), mk, priv, disk, second)
	if e != nil || stats.UploadedBlocks != 1 || stats.ReusedBlocks != 2 {
		t.Fatal("delta", stats, e)
	}
	current, e := NewReader(remote, mk, pub, disk, third)
	if e != nil {
		t.Fatal(e)
	}
	defer current.Close()
	got := make([]byte, len(data))
	if _, e = current.ReadAt(got, 0); e != nil || !bytes.Equal(got, data) {
		t.Fatal("full read mismatch", e)
	}
	if _, _, e = Publish(s, bytes.NewReader(data), uint64(len(data)), mk, priv, disk, first); !errors.Is(e, store.ErrConflict) {
		t.Fatal("stale parent not rejected", e)
	}
	head, _ := s.Head()
	if head != third {
		t.Fatal("conflict replaced head")
	}
	// Incomplete publication leaves the old head intact.
	m, e := s.Manifest(third)
	if e != nil {
		t.Fatal(e)
	}
	m.Parent = third
	m.Version++
	m.Entries = append([]protocol.ManifestEntry{}, m.Entries...)
	m.Entries[0].BlockID = protocol.BlockID{99}
	m, e = manifest.Build(m)
	if e != nil {
		t.Fatal(e)
	}
	m, _ = manifest.Sign(priv, m)
	missing, e := s.PutManifest(pub, m)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Commit(pub, third, missing); e == nil {
		t.Fatal("committed missing block")
	}
	head, _ = s.Head()
	if head != third {
		t.Fatal("incomplete commit changed head")
	}
	// A fresh client must have an independently supplied pin; an old valid pin
	// intentionally reopens old content and does not pretend to discover freshness.
	if _, e = NewReader(remote, mk, pub, disk, protocol.Digest{}); e == nil {
		t.Fatal("missing pin accepted")
	}
	old, e := NewReader(remote, mk, pub, disk, first)
	if e != nil {
		t.Fatal(e)
	}
	old.Close()
	if _, e = old.ReadAt(p, 0); e == nil {
		t.Fatal("closed reader")
	}
	// Corrupt stored ciphertext; no bytes of that block may escape.
	victim := m.Entries[1].BlockID
	path := filepath.Join(root, "blocks", fmtID(victim))
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	raw[len(raw)-1] ^= 1
	if e = os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	bad, e := NewReader(s, mk, pub, disk, third)
	if e != nil {
		t.Fatal(e)
	}
	defer bad.Close()
	out := make([]byte, 10)
	n, e = bad.ReadAt(out, protocol.BlockSize)
	if n != 0 || e == nil {
		t.Fatal("corrupt block exposed", n, e)
	}
}

// TestCrashBeforeCommitLeavesHeadIntact simulates a process that is killed
// after durably uploading a new snapshot's blocks and signed manifest, but
// before Commit's head CAS runs. The only correctness requirement is that a
// restarted process still sees a complete old-or-new snapshot, never a torn
// one; orphaned-but-durable objects from the interrupted publish are benign
// and may still be committed later. See the publication/crash-consistency
// gates in docs/phase-1-plan.md.
func TestCrashBeforeCommitLeavesHeadIntact(t *testing.T) {
	root := t.TempDir()
	s, e := store.Open(root)
	if e != nil {
		t.Fatal(e)
	}
	mk, _ := keys.GenerateMasterKey()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	disk := protocol.DiskID{7}
	original := bytes.Repeat([]byte{5}, protocol.BlockSize+1)

	first, _, e := Publish(s, bytes.NewReader(original), uint64(len(original)), mk, priv, disk, protocol.Digest{})
	if e != nil {
		t.Fatal(e)
	}
	firstManifest, e := s.Manifest(first)
	if e != nil {
		t.Fatal(e)
	}

	// Hand-replicate the first half of a second Publish call (seal the
	// changed block, write it, build/sign/store the manifest) without ever
	// calling Commit -- standing in for a crash right before the head CAS.
	changed := append([]byte{}, original...)
	changed[0] ^= 1
	bk, e := keys.DeriveBlockKey(mk, disk, firstManifest.KeyEpoch)
	if e != nil {
		t.Fatal(e)
	}
	idk, e := keys.DeriveBlockIDKey(mk, disk, firstManifest.KeyEpoch)
	if e != nil {
		t.Fatal(e)
	}
	b, id, e := block.Seal(bk, idk, protocol.Context{DiskID: disk, KeyEpoch: firstManifest.KeyEpoch, Index: 0, Generation: firstManifest.Version + 1}, changed[:protocol.BlockSize])
	if e != nil {
		t.Fatal(e)
	}
	if e = s.PutBlock(id, b); e != nil {
		t.Fatal(e)
	}
	m := protocol.Manifest{Format: protocol.FormatVersion, Suite: protocol.Suite, DiskID: disk, KeyEpoch: firstManifest.KeyEpoch, Version: firstManifest.Version + 1, Parent: first, DiskBytes: uint64(len(changed)), BlockBytes: protocol.BlockSize,
		Entries: []protocol.ManifestEntry{{Index: 0, Generation: firstManifest.Version + 1, BlockID: id}, firstManifest.Entries[1]}}
	m, e = manifest.Build(m)
	if e != nil {
		t.Fatal(e)
	}
	m, e = manifest.Sign(priv, m)
	if e != nil {
		t.Fatal(e)
	}
	crashed, e := s.PutManifest(pub, m)
	if e != nil {
		t.Fatal(e)
	}
	// "Crash": s.Commit(pub, first, crashed) is never called here.

	if head, e := s.Head(); e != nil || head != first {
		t.Fatal("interrupted publish must not move head", head, e)
	}
	// Reopen the store fresh, as a restarted process would.
	reopened, e := store.Open(root)
	if e != nil {
		t.Fatal(e)
	}
	if head, e := reopened.Head(); e != nil || head != first {
		t.Fatal("restart must still see the last committed head", head, e)
	}
	r, e := NewReader(reopened, mk, pub, disk, first)
	if e != nil {
		t.Fatal(e)
	}
	got := make([]byte, len(original))
	if _, e = r.ReadAt(got, 0); e != nil || !bytes.Equal(got, original) {
		t.Fatal("old head no longer reads as a complete, unmodified old snapshot", e)
	}
	r.Close()
	// Restarting the writer and finishing the interrupted commit must still
	// work; the orphaned-but-durable objects become reachable only now.
	if e = reopened.Commit(pub, first, crashed); e != nil {
		t.Fatal("resuming the interrupted commit after restart must still succeed", e)
	}
	if head, e := reopened.Head(); e != nil || head != crashed {
		t.Fatal("resumed commit did not advance head", head, e)
	}
	r2, e := NewReader(reopened, mk, pub, disk, crashed)
	if e != nil {
		t.Fatal(e)
	}
	defer r2.Close()
	got2 := make([]byte, len(changed))
	if _, e = r2.ReadAt(got2, 0); e != nil || !bytes.Equal(got2, changed) {
		t.Fatal("resumed snapshot does not read back as the data sealed before the crash", e)
	}
}
func fmtID(id protocol.BlockID) string {
	const hex = "0123456789abcdef"
	p := make([]byte, 64)
	for i, b := range id {
		p[2*i] = hex[b>>4]
		p[2*i+1] = hex[b&15]
	}
	return string(p)
}
func TestCompetingWritersAndShortSource(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	mk, _ := keys.GenerateMasterKey()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	disk := protocol.DiskID{2}
	data := []byte("baseline")
	pin, _, e := Publish(s, bytes.NewReader(data), uint64(len(data)), mk, priv, disk, protocol.Digest{})
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(v byte) {
			defer wg.Done()
			_, _, e := Publish(s, bytes.NewReader([]byte{v}), 1, mk, priv, disk, pin)
			results <- e
		}(byte(i + 1))
	}
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for e := range results {
		if e == nil {
			wins++
		} else if errors.Is(e, store.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatal(wins, conflicts)
	}
	head, _ := s.Head()
	if _, _, e = Publish(s, bytes.NewReader(nil), 1, mk, priv, disk, head); e == nil {
		t.Fatal("short source accepted")
	}
	after, _ := s.Head()
	if head != after {
		t.Fatal("short source changed head")
	}
	m, _ := s.Manifest(head)
	if e = manifest.VerifyPinned(pub, m, head); e != nil {
		t.Fatal(e)
	}
}

func TestEmptySnapshotSurvivesStoreReopen(t *testing.T) {
	root := t.TempDir()
	s, e := store.Open(root)
	if e != nil {
		t.Fatal(e)
	}
	mk, _ := keys.GenerateMasterKey()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	disk := protocol.DiskID{4}
	pin, stats, e := Publish(s, bytes.NewReader(nil), 0, mk, priv, disk, protocol.Digest{})
	if e != nil || stats.UploadedBlocks != 0 {
		t.Fatal(stats, e)
	}
	reopened, e := store.Open(root)
	if e != nil {
		t.Fatal(e)
	}
	head, e := reopened.Head()
	if e != nil || head != pin {
		t.Fatal("head not persisted", e)
	}
	r, e := NewReader(reopened, mk, pub, disk, pin)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	if n, e := r.ReadAt(make([]byte, 1), 0); n != 0 || e != io.EOF {
		t.Fatal(n, e)
	}
	// A new signer cannot replace a previously signed chain.
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	m, _ := reopened.Manifest(pin)
	m.Parent = pin
	m.Version++
	m, _ = manifest.Build(m)
	m, _ = manifest.Sign(other, m)
	next, e := reopened.PutManifest(other.Public().(ed25519.PublicKey), m)
	if e != nil {
		t.Fatal(e)
	}
	if e = reopened.Commit(other.Public().(ed25519.PublicKey), pin, next); e == nil {
		t.Fatal("unauthorized signer replaced chain")
	}
	after, _ := reopened.Head()
	if after != pin {
		t.Fatal("bad signer changed head")
	}
}
