// Command sync-agent publishes an offline image to a local development store.
package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"shadowdevice/internal/devcli"
	"shadowdevice/internal/image"
	"shadowdevice/internal/protocol"
	"shadowdevice/internal/store"
)

func main() {
	if e := run(); e != nil {
		log.Fatal(e)
	}
}
func run() error {
	root := flag.String("store", "", "ciphertext directory (single disk)")
	identity := flag.String("identity", "", "owner identity file outside store")
	init := flag.Bool("init", false, "create owner and .reader identity files; no upload")
	input := flag.String("image", "", "offline/frozen raw disk image")
	parent := flag.String("parent", "", "expected parent digest, or 'none' for first snapshot")
	flag.Parse()
	if *root == "" || *identity == "" {
		return fmt.Errorf("-store and -identity required")
	}
	if e := devcli.Outside(*identity, *root); e != nil {
		return e
	}
	if *init {
		return devcli.Init(*identity, *root)
	}
	if *input == "" || *parent == "" {
		return fmt.Errorf("-image and -parent required; source must be offline/frozen")
	}
	id, e := devcli.Load(*identity, true)
	if e != nil {
		return e
	}
	var pin protocol.Digest
	if *parent != "none" {
		pin, e = devcli.ParseDigest(*parent)
		if e != nil {
			return e
		}
	}
	f, e := os.Open(*input)
	if e != nil {
		return e
	}
	defer f.Close()
	before, e := f.Stat()
	if e != nil {
		return e
	}
	if !before.Mode().IsRegular() {
		return fmt.Errorf("image must be a regular frozen file")
	}
	s, e := store.Open(*root)
	if e != nil {
		return e
	}
	digest, stats, e := image.Publish(s, f, uint64(before.Size()), id.MasterKey, ed25519.NewKeyFromSeed(id.SigningSeed), id.DiskID, pin)
	if e != nil {
		return e
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Manifest string
		Stats    image.Stats
	}{hex.EncodeToString(digest[:]), stats})
}
