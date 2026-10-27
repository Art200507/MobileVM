// Package devcli manages trusted-local demo identity files. They are not PIN
// recovery kits, Android hardware keys, or production credential storage.
package devcli

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"shadowdevice/internal/keys"
	"shadowdevice/internal/protocol"
	"shadowdevice/internal/store"
)

type Identity struct {
	DiskID      protocol.DiskID
	MasterKey   keys.MasterKey
	PublicKey   []byte
	SigningSeed []byte `json:",omitempty"`
}

func Init(path, root string) error {
	for _, candidate := range []string{path, path + ".reader"} {
		if _, err := os.Lstat(candidate); err == nil {
			return errors.New("identity: refusing to replace existing identity files")
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if e := Outside(path, root); e != nil {
		return e
	}
	if e := Outside(path+".reader", root); e != nil {
		return e
	}
	mk, e := keys.GenerateMasterKey()
	if e != nil {
		return e
	}
	pub, priv, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		return e
	}
	id := Identity{MasterKey: mk, PublicKey: pub, SigningSeed: priv.Seed()}
	if _, e = rand.Read(id.DiskID[:]); e != nil {
		return e
	}
	// Create reader first; failures never overwrite either identity file.
	reader := id
	reader.SigningSeed = nil
	if e = writeNew(path+".reader", reader); e != nil {
		return e
	}
	return writeNew(path, id)
}
func writeNew(path string, v any) error {
	p, e := json.Marshal(v)
	if e != nil {
		return e
	}
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	defer f.Close()
	if _, e = f.Write(p); e != nil {
		return e
	}
	return f.Sync()
}
func Load(path string, writer bool) (Identity, error) {
	var id Identity
	info, e := os.Stat(path)
	if e != nil {
		return id, e
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return id, errors.New("identity: require regular file with permissions 0600")
	}
	p, e := store.ReadBounded(path, 4096)
	if e != nil {
		return id, e
	}
	if e = json.Unmarshal(p, &id); e != nil {
		return id, e
	}
	if id.DiskID == (protocol.DiskID{}) || len(id.PublicKey) != ed25519.PublicKeySize {
		return id, errors.New("identity: invalid identity")
	}
	if writer {
		if len(id.SigningSeed) != ed25519.SeedSize {
			return id, errors.New("identity: writer seed required")
		}
		priv := ed25519.NewKeyFromSeed(id.SigningSeed)
		if !priv.Public().(ed25519.PublicKey).Equal(ed25519.PublicKey(id.PublicKey)) {
			return id, errors.New("identity: inconsistent signer")
		}
	}
	return id, nil
}

// Outside resolves existing parent symlinks to keep secret files out of the
// served ciphertext directory. Local trusted users must not race path changes.
func Outside(path, root string) error {
	a, e := resolve(path)
	if e != nil {
		return e
	}
	b, e := resolve(root)
	if e != nil {
		return e
	}
	rel, e := filepath.Rel(b, a)
	if e != nil {
		return e
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return errors.New("identity: secret files must be outside ciphertext store")
	}
	return nil
}
func resolve(path string) (string, error) {
	abs, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	resolved, e := filepath.EvalSymlinks(abs)
	if e == nil {
		return resolved, nil
	}
	if !os.IsNotExist(e) {
		return "", e
	}
	parent, e := filepath.EvalSymlinks(filepath.Dir(abs))
	if e != nil {
		return "", e
	}
	return filepath.Join(parent, filepath.Base(abs)), nil
}
func ParseDigest(s string) (protocol.Digest, error) {
	var id protocol.Digest
	p, e := hex.DecodeString(s)
	if e != nil || len(p) != 32 {
		return id, errors.New("expected a 64-character manifest digest")
	}
	copy(id[:], p)
	return id, nil
}
