// Package compress is a bounded, stdlib-only compression primitive for disk
// blocks. It is a standalone building block, not yet wired into
// internal/block's fixed-size pipeline: doing that requires a tiered output
// size decision (storing a compressed block always costs the same as an
// incompressible one unless sizes are allowed to vary across a small set of
// fixed tiers), which is a deliberate wire-format choice, not a side effect
// of adding this package. See docs/product-expansion.md section 2.
package compress

import (
	"bytes"
	"compress/flate"
	"errors"
	"io"
)

// ErrTooLarge means decompression would exceed the caller-supplied bound.
var ErrTooLarge = errors.New("compress: decompressed output exceeds bound")

// Compress returns the DEFLATE-compressed form of p at the given level (see
// compress/flate for valid levels; flate.BestCompression is a reasonable
// default for data written once and read many times). The result carries no
// length framing of its own; a caller that pads the result to a fixed size
// relies on Decompress stopping at the stream's own end-of-data marker.
func Compress(p []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	w, e := flate.NewWriter(&buf, level)
	if e != nil {
		return nil, e
	}
	if _, e = w.Write(p); e != nil {
		return nil, e
	}
	if e = w.Close(); e != nil {
		return nil, e
	}
	return buf.Bytes(), nil
}

// Decompress inflates p, stopping at the DEFLATE stream's own end-of-data
// marker -- any bytes in p after that point (such as fixed-size padding)
// are never read. It never allocates or returns more than maxOutput bytes
// of plaintext, rejecting oversized or malformed input instead.
//
// This bound is a resource-usage safeguard, not an authentication check: p
// must already be authenticated by the caller (per this codebase's model of
// never acting on unauthenticated bytes) before its content is trusted.
func Decompress(p []byte, maxOutput int) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(p))
	defer r.Close()
	out, e := io.ReadAll(io.LimitReader(r, int64(maxOutput)+1))
	if e != nil {
		return nil, e
	}
	if len(out) > maxOutput {
		return nil, ErrTooLarge
	}
	return out, nil
}
