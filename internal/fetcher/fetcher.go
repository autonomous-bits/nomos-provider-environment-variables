// Package fetcher provides environment variable retrieval with caching.
package fetcher

import (
	"errors"
	"os"
	"sync"
)

var (
	// ErrNotFound is returned when an environment variable does not exist.
	ErrNotFound = errors.New("environment variable not found")
	// ErrValueTooLarge is returned when an environment variable value exceeds the maximum size.
	ErrValueTooLarge = errors.New("environment variable value too large")
)

// MaxValueSize is the maximum allowed size for an environment variable value (1MB).
const MaxValueSize = 1 * 1024 * 1024

// Fetcher retrieves environment variables with caching support.
type Fetcher struct {
	cache sync.Map
}

// New creates a new Fetcher instance.
func New() *Fetcher {
	return &Fetcher{}
}

// Fetch retrieves an environment variable by name, using cache if available.
func (f *Fetcher) Fetch(varName string) (string, error) {
	if cached, ok := f.cache.Load(varName); ok {
		return cached.(string), nil
	}
	value, exists := os.LookupEnv(varName)
	if !exists {
		return "", ErrNotFound
	}
	if len(value) > MaxValueSize {
		return "", ErrValueTooLarge
	}
	f.cache.Store(varName, value)
	return value, nil
}

// Clear removes all cached environment variable values.
func (f *Fetcher) Clear() {
	f.cache.Range(func(key, _ interface{}) bool {
		f.cache.Delete(key)
		return true
	})
}
