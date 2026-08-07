package main

import (
	"crypto/rand"
	"encoding/hex"
	"path"
	"strings"
)

// newRunID returns a random 16-byte hex identifier for a run.
func newRunID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand should never fail; fall back to a fixed marker rather than panic.
		return "0000000000000000"
	}
	return hex.EncodeToString(b)
}

// shellQuote single-quotes a string for safe use in a POSIX shell command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// dirOf returns the parent directory of a path (defaulting to "." if none).
func dirOf(p string) string {
	d := path.Dir(p)
	if d == "" {
		return "."
	}
	return d
}
