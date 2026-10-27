package image

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// RangeHandler exposes authenticated plaintext to trusted LOCAL VM tools only.
// It requires bounded single byte ranges, authenticating a range before sending
// response headers. It is not a protected-host interface or a production API.
func (r *Reader) RangeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Type", "application/octet-stream")
		if req.URL.Path != "/disk.raw" {
			http.NotFound(w, req)
			return
		}
		if req.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.FormatInt(r.Size(), 10))
			w.WriteHeader(http.StatusOK)
			return
		}
		if req.Method != http.MethodGet {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "read-only", http.StatusMethodNotAllowed)
			return
		}
		value := req.Header.Get("Range")
		start, end, ok := strings.Cut(strings.TrimPrefix(value, "bytes="), "-")
		a, e1 := strconv.ParseInt(start, 10, 64)
		b, e2 := strconv.ParseInt(end, 10, 64)
		if !strings.HasPrefix(value, "bytes=") || !ok || e1 != nil || e2 != nil || a < 0 || b < a || b >= r.Size() || b-a >= 32<<20 {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", r.Size()))
			http.Error(w, "require a single bounded byte range of at most 32 MiB", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		p := make([]byte, int(b-a+1))
		defer clear(p)
		if _, e := r.ReadAt(p, a); e != nil {
			http.Error(w, "disk read failed", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(p)))
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", a, b, r.Size()))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(p)
	})
}
