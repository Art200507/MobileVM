// Package store provides a single-disk, local development ciphertext store.
// Local filesystem integrity and fsync semantics are trusted. HTTP is read-only.
package store

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"shadowdevice/internal/block"
	"shadowdevice/internal/manifest"
	"shadowdevice/internal/protocol"
)

const MaxManifestBytes = 16 << 20

var ErrConflict = errors.New("store: stale parent or incompatible writer")

type Store struct{ root string }

func Open(root string) (*Store, error) {
	abs, e := filepath.Abs(root)
	if e != nil {
		return nil, e
	}
	for _, d := range []string{abs, filepath.Join(abs, "blocks"), filepath.Join(abs, "manifests")} {
		if e = os.MkdirAll(d, 0700); e != nil {
			return nil, e
		}
	}
	for _, dir := range []string{filepath.Join(abs, "blocks"), filepath.Join(abs, "manifests"), abs, filepath.Dir(abs)} {
		if e := syncDir(dir); e != nil {
			return nil, e
		}
	}
	return &Store{abs}, nil
}
func name(id [32]byte) string { return hex.EncodeToString(id[:]) }

// immutable publishes only fully synced data, never overwriting an object.
func (s *Store) immutable(dir, n string, p []byte) error {
	dst := filepath.Join(s.root, dir, n)
	if old, e := os.ReadFile(dst); e == nil {
		if bytes.Equal(old, p) {
			return nil
		}
		return errors.New("store: immutable object collision")
	} else if !os.IsNotExist(e) {
		return e
	}
	f, e := os.CreateTemp(filepath.Join(s.root, dir), ".upload-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(p); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Link(f.Name(), dst); e != nil {
		if os.IsExist(e) {
			old, re := os.ReadFile(dst)
			if re == nil && bytes.Equal(old, p) {
				return nil
			}
		}
		return e
	}
	return syncDir(filepath.Dir(dst))
}
func syncDir(p string) error {
	f, e := os.Open(p)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
func ReadBounded(p string, max int64) ([]byte, error) {
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return ReadLimit(f, max)
}
func ReadLimit(r io.Reader, max int64) ([]byte, error) {
	p, e := io.ReadAll(io.LimitReader(r, max+1))
	if e != nil {
		return nil, e
	}
	if int64(len(p)) > max {
		return nil, errors.New("store: input too large")
	}
	return p, nil
}
func DecodeManifest(p []byte) (protocol.Manifest, error) {
	var m protocol.Manifest
	if len(p) > MaxManifestBytes {
		return m, errors.New("store: manifest too large")
	}
	d := json.NewDecoder(bytes.NewReader(p))
	d.DisallowUnknownFields()
	if e := d.Decode(&m); e != nil {
		return m, e
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return m, errors.New("store: trailing manifest data")
	}
	return m, nil
}
func (s *Store) PutBlock(id protocol.BlockID, b protocol.EncryptedBlock) error {
	p, e := block.Marshal(b)
	if e != nil {
		return e
	}
	return s.immutable("blocks", name(id), p)
}
func (s *Store) Block(id protocol.BlockID) (protocol.EncryptedBlock, error) {
	p, e := ReadBounded(filepath.Join(s.root, "blocks", name(id)), block.ObjectSize)
	if e != nil {
		return protocol.EncryptedBlock{}, e
	}
	return block.Unmarshal(p)
}
func (s *Store) PutManifest(pub ed25519.PublicKey, m protocol.Manifest) (protocol.Digest, error) {
	var zero protocol.Digest
	if e := manifest.Verify(pub, m); e != nil {
		return zero, e
	}
	p, e := json.Marshal(m)
	if e != nil {
		return zero, e
	}
	if len(p) > MaxManifestBytes {
		return zero, errors.New("store: manifest too large")
	}
	id := manifest.Digest(m)
	return id, s.immutable("manifests", name(id), p)
}
func (s *Store) Manifest(id protocol.Digest) (protocol.Manifest, error) {
	p, e := ReadBounded(filepath.Join(s.root, "manifests", name(id)), MaxManifestBytes)
	if e != nil {
		return protocol.Manifest{}, e
	}
	m, e := DecodeManifest(p)
	if e == nil && manifest.Digest(m) != id {
		return m, errors.New("store: wrong manifest digest")
	}
	return m, e
}

// Head is discovery only. Clients still require independent signer/checkpoint trust.
func (s *Store) Head() (protocol.Digest, error) {
	var id protocol.Digest
	p, e := ReadBounded(filepath.Join(s.root, "head"), 32)
	if os.IsNotExist(e) {
		return id, nil
	}
	if e != nil {
		return id, e
	}
	if len(p) != 32 {
		return id, errors.New("store: corrupt head")
	}
	copy(id[:], p)
	return id, nil
}

// Commit serializes local writers across processes. The directory is single-disk;
// the current manifest must verify with the same pinned signer and disk identity.
func (s *Store) Commit(pub ed25519.PublicKey, expected, next protocol.Digest) error {
	f, e := os.OpenFile(filepath.Join(s.root, ".writer-lock"), os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	if e = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); e != nil {
		return e
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	head, e := s.Head()
	if e != nil {
		return e
	}
	if head != expected {
		return ErrConflict
	}
	m, e := s.Manifest(next)
	if e != nil {
		return e
	}
	if e = manifest.Verify(pub, m); e != nil {
		return e
	}
	if m.Parent != expected {
		return ErrConflict
	}
	if expected == (protocol.Digest{}) {
		if m.Version != 1 {
			return ErrConflict
		}
	} else {
		old, e := s.Manifest(expected)
		if e != nil {
			return e
		}
		if e = manifest.Verify(pub, old); e != nil {
			return e
		}
		if old.Version == ^uint64(0) || m.Version != old.Version+1 || m.DiskID != old.DiskID || m.KeyEpoch != old.KeyEpoch {
			return ErrConflict
		}
	}
	for _, entry := range m.Entries {
		b, e := s.Block(entry.BlockID)
		if e != nil {
			return fmt.Errorf("store: missing/invalid block: %w", e)
		}
		if b.Context != (protocol.Context{DiskID: m.DiskID, KeyEpoch: m.KeyEpoch, Index: entry.Index, Generation: entry.Generation}) {
			return errors.New("store: block context mismatch")
		}
	}
	temp, e := os.CreateTemp(s.root, ".head-")
	if e != nil {
		return e
	}
	defer os.Remove(temp.Name())
	if _, e = temp.Write(next[:]); e != nil {
		temp.Close()
		return e
	}
	if e = temp.Sync(); e != nil {
		temp.Close()
		return e
	}
	if e = temp.Close(); e != nil {
		return e
	}
	if e = os.Rename(temp.Name(), filepath.Join(s.root, "head")); e != nil {
		return e
	}
	return syncDir(s.root)
}
