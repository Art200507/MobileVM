package store

import (
	"bytes"
	"shadowdevice/internal/block"
	"shadowdevice/internal/keys"
	"shadowdevice/internal/protocol"
	"testing"
)

func TestImmutableObjects(t *testing.T) {
	s, e := Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	mk, _ := keys.GenerateMasterKey()
	disk := protocol.DiskID{1}
	bk, _ := keys.DeriveBlockKey(mk, disk, 1)
	ik, _ := keys.DeriveBlockIDKey(mk, disk, 1)
	c := protocol.Context{DiskID: disk, KeyEpoch: 1, Generation: 1}
	b, id, e := block.Seal(bk, ik, c, make([]byte, protocol.BlockSize))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.PutBlock(id, b); e != nil {
		t.Fatal(e)
	}
	if e = s.PutBlock(id, b); e != nil {
		t.Fatal("retry not idempotent", e)
	}
	other, _, e := block.Seal(bk, ik, c, bytes.Repeat([]byte{1}, protocol.BlockSize))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.PutBlock(id, other); e == nil {
		t.Fatal("immutable ID overwritten")
	}
	got, e := s.Block(id)
	if e != nil {
		t.Fatal(e)
	}
	plain, e := block.Open(bk, ik, c, id, got)
	if e != nil || !bytes.Equal(plain, make([]byte, protocol.BlockSize)) {
		t.Fatal("original object changed", e)
	}
}
