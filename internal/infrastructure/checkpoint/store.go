// Package checkpoint persists completed expensive parsing and embedding work.
package checkpoint

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// Key identifies the exact inputs whose result can replace another provider call.
func Key(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func Path(kind, key string) string {
	root := os.Getenv("WEKNORA_PROCESSING_CHECKPOINT_DIR")
	if root == "" {
		return ""
	}
	return filepath.Join(root, kind, key+".json.gz")
}
func Load(kind, key string, dst any) bool {
	p := Path(kind, key)
	if p == "" {
		return false
	}
	f, e := os.Open(p)
	if e != nil {
		return false
	}
	defer f.Close()
	z, e := gzip.NewReader(f)
	if e != nil {
		return false
	}
	defer z.Close()
	return json.NewDecoder(z).Decode(dst) == nil
}
func Save(kind, key string, value any) error {
	p := Path(kind, key)
	if p == "" {
		return nil
	}
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(p), ".checkpoint-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	z := gzip.NewWriter(f)
	e = json.NewEncoder(z).Encode(value)
	if ce := z.Close(); e == nil {
		e = ce
	}
	if ce := f.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(f.Name(), p)
}
