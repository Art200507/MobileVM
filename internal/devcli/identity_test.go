package devcli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestIdentitySeparationAndPermissions(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "ciphertext")
	path := filepath.Join(dir, "owner.json")
	if e := Init(path, root); e != nil {
		t.Fatal(e)
	}
	owner, e := Load(path, true)
	if e != nil {
		t.Fatal(e)
	}
	reader, e := Load(path+".reader", false)
	if e != nil {
		t.Fatal(e)
	}
	if len(reader.SigningSeed) != 0 || len(owner.SigningSeed) != 32 || reader.MasterKey != owner.MasterKey || !bytes.Equal(reader.PublicKey, owner.PublicKey) {
		t.Fatal("bad identity separation")
	}
	before, _ := os.ReadFile(path)
	if e = Init(path, root); e == nil {
		t.Fatal("overwrote identity")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("identity modified")
	}
	if e = Outside(filepath.Join(root, "secret"), root); e == nil {
		t.Fatal("allowed secret under store")
	}
	os.Chmod(path, 0644)
	if _, e = Load(path, true); e == nil {
		t.Fatal("accepted publicly readable identity")
	}
	if _, e = Load(path+".reader", true); e == nil {
		t.Fatal("reader can sign")
	}
}
func TestStoreSymlinkIsolation(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "store")
	os.Mkdir(root, 0700)
	alias := filepath.Join(dir, "alias")
	if e := os.Symlink(root, alias); e != nil {
		t.Fatal(e)
	}
	if Outside(filepath.Join(alias, "secret"), root) == nil {
		t.Fatal("symlink bypass")
	}
}
