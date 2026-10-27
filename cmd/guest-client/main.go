// Command guest-client reads an explicitly pinned snapshot on a trusted client.
// It is not a protected Android guest or a PIN recovery implementation.
package main

import (
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"shadowdevice/internal/devcli"
	"shadowdevice/internal/image"
	"shadowdevice/internal/store"
)

func main() {
	if e := run(); e != nil {
		log.Fatal(e)
	}
}
func run() error {
	root := flag.String("store", "", "local ciphertext directory")
	remote := flag.String("url", "", "read-only development HTTP(S) origin")
	identity := flag.String("identity", "", "trusted .reader identity file")
	pin := flag.String("manifest", "", "independently trusted manifest digest")
	offset := flag.Int64("offset", 0, "byte offset")
	length := flag.Int64("length", 4096, "bytes to read (maximum 64 MiB)")
	output := flag.String("out", "", "new plaintext output file; '-' for stdout")
	listen := flag.String("listen", "", "serve plaintext byte ranges at loopback /disk.raw for trusted VM tools instead of exporting a range")
	flag.Parse()
	if *identity == "" || *pin == "" || (*output == "" && *listen == "") || (*root == "") == (*remote == "") {
		return fmt.Errorf("require -identity, -manifest, one of -out/-listen and exactly one of -store/-url")
	}
	if *listen != "" && *output != "" {
		return fmt.Errorf("choose -listen or -out")
	}
	if *offset < 0 || *length < 0 || *length > 64<<20 {
		return fmt.Errorf("invalid offset/length")
	}
	id, e := devcli.Load(*identity, false)
	if e != nil {
		return e
	}
	if len(id.SigningSeed) != 0 {
		return fmt.Errorf("use the .reader identity, not the signing identity")
	}
	digest, e := devcli.ParseDigest(*pin)
	if e != nil {
		return e
	}
	var src image.Source
	if *root != "" {
		if e = devcli.Outside(*identity, *root); e != nil {
			return e
		}
		if *output != "" && *output != "-" {
			if e = devcli.Outside(*output, *root); e != nil {
				return fmt.Errorf("plaintext output must stay outside store: %w", e)
			}
		}
		src, e = store.Open(*root)
	} else {
		src, e = store.NewRemote(*remote)
	}
	if e != nil {
		return e
	}
	r, e := image.NewReader(src, id.MasterKey, ed25519.PublicKey(id.PublicKey), id.DiskID, digest)
	if e != nil {
		return e
	}
	defer r.Close()
	if *listen != "" {
		host, _, err := net.SplitHostPort(*listen)
		if err != nil {
			return err
		}
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return fmt.Errorf("plaintext disk server requires numeric loopback address")
		}
		fmt.Fprintf(os.Stderr, "Trusted-local plaintext disk endpoint: http://%s/disk.raw\n", *listen)
		server := http.Server{Addr: *listen, Handler: r.RangeHandler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
		return server.ListenAndServe()
	}
	if *offset > r.Size() || *length > r.Size()-*offset {
		return fmt.Errorf("requested range exceeds image")
	}
	// Authenticate the complete requested range before publishing any output.
	p := make([]byte, int(*length))
	defer clear(p)
	if _, e = r.ReadAt(p, *offset); e != nil {
		return e
	}
	var out io.Writer = os.Stdout
	if *output != "-" {
		f, e := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		defer f.Close()
		out = f
	}
	if _, e = out.Write(p); e != nil {
		return e
	}
	if f, ok := out.(*os.File); ok && f != os.Stdout {
		if e = f.Sync(); e != nil {
			return e
		}
	}
	return json.NewEncoder(os.Stderr).Encode(r.Stats())
}
