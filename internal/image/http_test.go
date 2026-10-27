package image

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"shadowdevice/internal/keys"
	"shadowdevice/internal/protocol"
	"shadowdevice/internal/store"
)

func TestPlaintextRangeEndpoint(t *testing.T) {
	dir := t.TempDir()
	s, _ := store.Open(dir)
	mk, _ := keys.GenerateMasterKey()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	disk := protocol.DiskID{3}
	data := bytes.Repeat([]byte("synthetic-data"), 100)
	pin, _, e := Publish(s, bytes.NewReader(data), uint64(len(data)), mk, priv, disk, protocol.Digest{})
	if e != nil {
		t.Fatal(e)
	}
	r, e := NewReader(s, mk, pub, disk, pin)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	h := r.RangeHandler()
	request := func(method, path, span string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Range", span)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	w := request("HEAD", "/disk.raw", "")
	if w.Code != 200 || w.Body.Len() != 0 || r.Stats().FetchedBlocks != 0 {
		t.Fatal("HEAD fetched data", w.Code)
	}
	w = request("GET", "/disk.raw", "bytes=10-31")
	if w.Code != 206 || !bytes.Equal(w.Body.Bytes(), data[10:32]) {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, span := range []string{"", "bytes=-20", "bytes=0-", "bytes=0-1,3-4", "bytes=0-999999", "bytes=-1-3", "items=0-3"} {
		if w = request("GET", "/disk.raw", span); w.Code != 416 {
			t.Fatal(span, w.Code)
		}
	}
	if w = request("PUT", "/disk.raw", ""); w.Code != 405 {
		t.Fatal(w.Code)
	}
	if w = request("GET", "/elsewhere", "bytes=0-1"); w.Code != 404 {
		t.Fatal(w.Code)
	}
	// A corrupt upstream block returns an error before a plaintext success reply.
	m, _ := s.Manifest(pin)
	path := filepath.Join(dir, "blocks", fmtID(m.Entries[0].BlockID))
	p, _ := os.ReadFile(path)
	p[len(p)-1] ^= 1
	os.WriteFile(path, p, 0600)
	bad, e := NewReader(s, mk, pub, disk, pin)
	if e != nil {
		t.Fatal(e)
	}
	defer bad.Close()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/disk.raw", nil)
	req.Header.Set("Range", "bytes=0-10")
	bad.RangeHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway || bytes.Contains(rec.Body.Bytes(), data[:11]) {
		t.Fatal("bad block served", rec.Code)
	}
}
func TestHTTPStoreRejectsWriteAndOversizedRemote(t *testing.T) {
	s, _ := store.Open(t.TempDir())
	server := httptest.NewServer(s.Handler())
	defer server.Close()
	req, _ := http.NewRequest("PUT", server.URL+"/blocks/"+fmtID(protocol.BlockID{}), bytes.NewReader([]byte("x")))
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 405 {
		t.Fatal(resp.StatusCode)
	}
	huge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(bytes.Repeat([]byte{0}, store.MaxManifestBytes+1))
	}))
	defer huge.Close()
	remote, _ := store.NewRemote(huge.URL)
	if _, e = remote.Manifest(protocol.Digest{1}); e == nil {
		t.Fatal("accepted oversized manifest")
	}
}
