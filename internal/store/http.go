package store

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"shadowdevice/internal/block"
	"shadowdevice/internal/manifest"
	"shadowdevice/internal/protocol"
)

// Handler serves ciphertext only. Bind to loopback; authentication, quotas and
// production transport policy are intentionally not provided by this demo.
func (s *Store) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /blocks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, e := parseID(r.PathValue("id"))
		if e != nil {
			http.Error(w, "invalid ID", 400)
			return
		}
		b, e := s.Block(protocol.BlockID(id))
		if e != nil {
			http.NotFound(w, r)
			return
		}
		p, e := block.Marshal(b)
		if e != nil {
			http.Error(w, "invalid object", 500)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(p)
	})
	mux.HandleFunc("GET /manifests/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, e := parseID(r.PathValue("id"))
		if e != nil {
			http.Error(w, "invalid ID", 400)
			return
		}
		m, e := s.Manifest(protocol.Digest(id))
		if e != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(m)
	})
	return mux
}
func parseID(s string) ([32]byte, error) {
	var id [32]byte
	p, e := hex.DecodeString(s)
	if e != nil || len(p) != 32 {
		return id, errors.New("invalid digest")
	}
	copy(id[:], p)
	return id, nil
}

type Remote struct {
	base   string
	client *http.Client
}

func NewRemote(base string) (*Remote, error) {
	u, e := url.Parse(base)
	if e != nil {
		return nil, e
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("store: expected HTTP(S) origin")
	}
	return &Remote{strings.TrimRight(base, "/"), &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("store: redirects disabled") }}}, nil
}
func (r *Remote) get(ctx context.Context, path string, limit int64) ([]byte, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", r.base+path, nil)
	if e != nil {
		return nil, e
	}
	resp, e := r.client.Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("store: HTTP %d", resp.StatusCode)
	}
	return ReadLimit(resp.Body, limit)
}
func (r *Remote) Block(id protocol.BlockID) (protocol.EncryptedBlock, error) {
	p, e := r.get(context.Background(), "/blocks/"+name(id), block.ObjectSize)
	if e != nil {
		return protocol.EncryptedBlock{}, e
	}
	return block.Unmarshal(p)
}
func (r *Remote) Manifest(id protocol.Digest) (protocol.Manifest, error) {
	p, e := r.get(context.Background(), "/manifests/"+name(id), MaxManifestBytes)
	if e != nil {
		return protocol.Manifest{}, e
	}
	m, e := DecodeManifest(p)
	if e == nil && name(manifest.Digest(m)) != name(id) {
		return m, errors.New("store: wrong manifest digest")
	}
	return m, e
}
