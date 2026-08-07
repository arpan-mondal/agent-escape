package main

import (
	"fmt"
	"sort"
	"strings"
)

// backends is the registry of available isolation backends, keyed by name.
var backends = map[string]Backend{}

// registerBackend adds a backend to the registry (called from init()).
func registerBackend(b Backend) {
	backends[b.Name()] = b
}

// getBackend looks up a backend by name.
func getBackend(name string) (Backend, error) {
	b, ok := backends[name]
	if !ok {
		return nil, fmt.Errorf("unknown backend %q (available: %s)", name, availableBackends())
	}
	return b, nil
}

// availableBackends returns a sorted, comma-separated list of registered backends.
func availableBackends() string {
	names := make([]string, 0, len(backends))
	for n := range backends {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
