// Package image publishes immutable snapshots and reads verified blocks on demand.
// All cryptography runs in the trusted development process, not an Android pVM.
package image

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"io"
	"os"
	"sync"

	"shadowdevice/internal/block"
	"shadowdevice/internal/keys"
	"shadowdevice/internal/manifest"
	"shadowdevice/internal/protocol"
	"shadowdevice/internal/store"
)

type Source interface {
	Block(protocol.BlockID) (protocol.EncryptedBlock, error)
	Manifest(protocol.Digest) (protocol.Manifest, error)
}
type Stats struct {
	FetchedBlocks  uint64
	FetchedBytes   uint64
	UploadedBlocks uint64
	ReusedBlocks   uint64
}

func context(m protocol.Manifest, e protocol.ManifestEntry) protocol.Context {
	return protocol.Context{DiskID: m.DiskID, KeyEpoch: m.KeyEpoch, Index: e.Index, Generation: e.Generation}
}

// Publish requires a stable source (an offline image or already frozen snapshot).
// It does not create a filesystem/application-consistent snapshot of a live VM.
func Publish(s *store.Store, src io.ReaderAt, size uint64, mk keys.MasterKey, priv ed25519.PrivateKey, disk protocol.DiskID, parent protocol.Digest) (protocol.Digest, Stats, error) {
	var stats Stats
	var before os.FileInfo
	if f, ok := src.(interface{ Stat() (os.FileInfo, error) }); ok {
		var err error
		before, err = f.Stat()
		if err != nil {
			return protocol.Digest{}, stats, err
		}
		if before.Size() < 0 || uint64(before.Size()) != size {
			return protocol.Digest{}, stats, errors.New("image: source size mismatch")
		}
	}
	var zero protocol.Digest
	if len(priv) != ed25519.PrivateKeySize || disk == (protocol.DiskID{}) || size > uint64(protocol.MaxBlocks)*protocol.BlockSize {
		return zero, stats, errors.New("image: invalid input")
	}
	pub := priv.Public().(ed25519.PublicKey)
	version := uint64(1)
	epoch := uint64(1)
	var previous protocol.Manifest
	if parent != (protocol.Digest{}) {
		var e error
		previous, e = s.Manifest(parent)
		if e != nil {
			return zero, stats, e
		}
		if e = manifest.VerifyPinned(pub, previous, parent); e != nil {
			return zero, stats, e
		}
		if previous.DiskID != disk || previous.Version == ^uint64(0) {
			return zero, stats, errors.New("image: wrong disk or exhausted version")
		}
		version = previous.Version + 1
		epoch = previous.KeyEpoch
	}
	bk, e := keys.DeriveBlockKey(mk, disk, epoch)
	if e != nil {
		return zero, stats, e
	}
	idk, e := keys.DeriveBlockIDKey(mk, disk, epoch)
	if e != nil {
		return zero, stats, e
	}
	defer clear(bk[:])
	defer clear(idk[:])
	m := protocol.Manifest{Format: protocol.FormatVersion, Suite: protocol.Suite, DiskID: disk, KeyEpoch: epoch, Version: version, Parent: parent, DiskBytes: size, BlockBytes: protocol.BlockSize}
	buf := make([]byte, protocol.BlockSize)
	defer clear(buf)
	for offset := uint64(0); offset < size; offset += protocol.BlockSize {
		clear(buf)
		n := min(uint64(protocol.BlockSize), size-offset)
		got, re := src.ReadAt(buf[:n], int64(offset))
		if got != int(n) || (re != nil && re != io.EOF) {
			if re == nil {
				re = io.ErrUnexpectedEOF
			}
			return zero, stats, re
		}
		index := offset / protocol.BlockSize
		if index < uint64(len(previous.Entries)) {
			entry := previous.Entries[index]
			old, re := s.Block(entry.BlockID)
			if re != nil {
				return zero, stats, re
			}
			plain, re := block.Open(bk, idk, context(previous, entry), entry.BlockID, old)
			if re != nil {
				return zero, stats, re
			}
			same := bytes.Equal(plain, buf)
			clear(plain)
			if same {
				m.Entries = append(m.Entries, entry)
				stats.ReusedBlocks++
				continue
			}
		}
		b, id, re := block.Seal(bk, idk, protocol.Context{DiskID: disk, KeyEpoch: epoch, Index: index, Generation: version}, buf)
		if re != nil {
			return zero, stats, re
		}
		if re = s.PutBlock(id, b); re != nil {
			return zero, stats, re
		}
		m.Entries = append(m.Entries, protocol.ManifestEntry{Index: index, Generation: version, BlockID: id})
		stats.UploadedBlocks++
	}
	if before != nil {
		after, err := src.(interface{ Stat() (os.FileInfo, error) }).Stat()
		if err != nil {
			return zero, stats, err
		}
		if before.Size() != after.Size() || before.ModTime() != after.ModTime() {
			return zero, stats, errors.New("image: source changed before publication")
		}
	}
	m, e = manifest.Build(m)
	if e != nil {
		return zero, stats, e
	}
	m, e = manifest.Sign(priv, m)
	if e != nil {
		return zero, stats, e
	}
	id, e := s.PutManifest(pub, m)
	if e != nil {
		return zero, stats, e
	}
	if e = s.Commit(pub, parent, id); e != nil {
		return zero, stats, e
	}
	return id, stats, nil
}

// Reader keeps at most one decrypted block. It is safe for concurrent ReadAt.
// Close clears owned buffers only, not all runtime copies or previously returned data.
type Reader struct {
	mu     sync.Mutex
	source Source
	m      protocol.Manifest
	bk     keys.BlockKey
	idk    keys.BlockIDKey
	cached []byte
	index  uint64
	stats  Stats
	closed bool
}

func NewReader(s Source, mk keys.MasterKey, pub ed25519.PublicKey, disk protocol.DiskID, pin protocol.Digest) (*Reader, error) {
	if pin == (protocol.Digest{}) {
		return nil, errors.New("image: independent manifest pin required")
	}
	m, e := s.Manifest(pin)
	if e != nil {
		return nil, e
	}
	if e = manifest.VerifyPinned(pub, m, pin); e != nil {
		return nil, e
	}
	if m.DiskID != disk {
		return nil, errors.New("image: wrong disk")
	}
	bk, e := keys.DeriveBlockKey(mk, disk, m.KeyEpoch)
	if e != nil {
		return nil, e
	}
	idk, e := keys.DeriveBlockIDKey(mk, disk, m.KeyEpoch)
	if e != nil {
		return nil, e
	}
	m.Entries = append([]protocol.ManifestEntry{}, m.Entries...)
	return &Reader{source: s, m: m, bk: bk, idk: idk}, nil
}
func (r *Reader) Size() int64  { return int64(r.m.DiskBytes) }
func (r *Reader) Stats() Stats { r.mu.Lock(); defer r.mu.Unlock(); return r.stats }
func (r *Reader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	clear(r.bk[:])
	clear(r.idk[:])
	clear(r.cached)
	r.cached = nil
	r.closed = true
	return nil
}
func (r *Reader) ReadAt(p []byte, off int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return 0, errors.New("image: reader closed")
	}
	if off < 0 {
		return 0, errors.New("image: negative offset")
	}
	if len(p) == 0 {
		return 0, nil
	}
	if uint64(off) >= r.m.DiskBytes {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) && uint64(off) < r.m.DiskBytes {
		index := uint64(off) / protocol.BlockSize
		entry := r.m.Entries[index]
		if r.cached == nil || r.index != index {
			b, e := r.source.Block(entry.BlockID)
			if e != nil {
				return n, e
			}
			plain, e := block.Open(r.bk, r.idk, context(r.m, entry), entry.BlockID, b)
			if e != nil {
				return n, e
			}
			clear(r.cached)
			r.cached = plain
			r.index = index
			r.stats.FetchedBlocks++
			r.stats.FetchedBytes += uint64(block.ObjectSize)
		}
		within := uint64(off) % protocol.BlockSize
		amount := min(uint64(len(p)-n), uint64(protocol.BlockSize)-within, r.m.DiskBytes-uint64(off))
		copy(p[n:n+int(amount)], r.cached[within:within+amount])
		n += int(amount)
		off += int64(amount)
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
