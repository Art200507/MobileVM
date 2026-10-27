// Package block seals immutable, fixed-size disk blocks. Decryption requires
// expectations from an independently verified manifest.
package block

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"

	"shadowdevice/internal/keys"
	"shadowdevice/internal/protocol"
)

const ObjectSize = 4 + 16 + 8 + 8 + 8 + 32 + protocol.BlockSize + 28

var ErrInvalid = errors.New("block: invalid or unauthenticated object")

func header(b protocol.EncryptedBlock) []byte {
	p := []byte("shadowdevice/block/v1\x00")
	p = binary.BigEndian.AppendUint32(p, b.Format)
	p = append(p, b.Context.DiskID[:]...)
	p = binary.BigEndian.AppendUint64(p, b.Context.KeyEpoch)
	p = binary.BigEndian.AppendUint64(p, b.Context.Index)
	p = binary.BigEndian.AppendUint64(p, b.Context.Generation)
	return append(p, b.Salt[:]...)
}
func aead(k keys.BlockKey, salt [32]byte) (cipher.AEAD, error) {
	raw, e := hkdf.Key(sha256.New, k[:], salt[:], "shadowdevice/v1/object-aes-key", 32)
	if e != nil {
		return nil, e
	}
	defer clear(raw)
	c, e := aes.NewCipher(raw)
	if e != nil {
		return nil, e
	}
	return cipher.NewGCMWithRandomNonce(c)
}
func validContext(c protocol.Context) bool {
	return c.DiskID != (protocol.DiskID{}) && c.KeyEpoch > 0 && c.Generation > 0 && c.Index < protocol.MaxBlocks
}
func Seal(k keys.BlockKey, idk keys.BlockIDKey, c protocol.Context, p []byte) (protocol.EncryptedBlock, protocol.BlockID, error) {
	b := protocol.EncryptedBlock{Format: protocol.FormatVersion, Context: c}
	if len(p) != protocol.BlockSize || !validContext(c) {
		return b, protocol.BlockID{}, ErrInvalid
	}
	if _, e := rand.Read(b.Salt[:]); e != nil {
		return b, protocol.BlockID{}, e
	}
	a, e := aead(k, b.Salt)
	if e != nil {
		return b, protocol.BlockID{}, e
	}
	b.Sealed = a.Seal(nil, nil, p, header(b))
	id, e := DeriveID(idk, b)
	return b, id, e
}
func DeriveID(k keys.BlockIDKey, b protocol.EncryptedBlock) (protocol.BlockID, error) {
	var id protocol.BlockID
	if b.Format != protocol.FormatVersion || !validContext(b.Context) || len(b.Sealed) != protocol.BlockSize+28 {
		return id, ErrInvalid
	}
	h := hmac.New(sha256.New, k[:])
	h.Write(header(b))
	h.Write(b.Sealed)
	copy(id[:], h.Sum(nil))
	return id, nil
}
func Open(k keys.BlockKey, idk keys.BlockIDKey, expected protocol.Context, id protocol.BlockID, b protocol.EncryptedBlock) ([]byte, error) {
	if b.Context != expected {
		return nil, ErrInvalid
	}
	actual, e := DeriveID(idk, b)
	if e != nil || !hmac.Equal(actual[:], id[:]) {
		return nil, ErrInvalid
	}
	a, e := aead(k, b.Salt)
	if e != nil {
		return nil, e
	}
	p, e := a.Open(nil, nil, b.Sealed, header(b))
	if e != nil {
		return nil, ErrInvalid
	}
	return p, nil
}

// Marshal uses fixed-width big-endian fields. No JSON encoding is used for blocks.
func Marshal(b protocol.EncryptedBlock) ([]byte, error) {
	if _, e := DeriveID(keys.BlockIDKey{}, b); e != nil {
		return nil, e
	}
	h := header(b)
	out := append([]byte{}, h[len("shadowdevice/block/v1\x00"):]...)
	return append(out, b.Sealed...), nil
}
func Unmarshal(p []byte) (protocol.EncryptedBlock, error) {
	var b protocol.EncryptedBlock
	if len(p) != ObjectSize {
		return b, ErrInvalid
	}
	b.Format = binary.BigEndian.Uint32(p[:4])
	copy(b.Context.DiskID[:], p[4:20])
	b.Context.KeyEpoch = binary.BigEndian.Uint64(p[20:28])
	b.Context.Index = binary.BigEndian.Uint64(p[28:36])
	b.Context.Generation = binary.BigEndian.Uint64(p[36:44])
	copy(b.Salt[:], p[44:76])
	b.Sealed = append([]byte{}, p[76:]...)
	if _, e := DeriveID(keys.BlockIDKey{}, b); e != nil {
		return protocol.EncryptedBlock{}, e
	}
	return b, nil
}
