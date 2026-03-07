// Package fetcher provides environment variable retrieval with caching.
package fetcher

import (
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/autonomous-bits/nomos-provider-environment-variables/internal/logger"
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
	cache  sync.Map
	logger *logger.Logger
}

// New creates a new Fetcher instance.
func New() *Fetcher {
	return &Fetcher{}
}

// NewWithLogger creates a new Fetcher instance with a logger.
func NewWithLogger(log *logger.Logger) *Fetcher {
	return &Fetcher{logger: log}
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

// FetchAll returns all environment variables matching the given prefix.
// The prefix is stripped from each returned key.
// If matchPrefix is empty, all environment variables are returned.
// Entries with values exceeding MaxValueSize are skipped with a warning log.
// Keys that exactly equal the prefix (resulting in an empty relative key) are skipped.
func (f *Fetcher) FetchAll(matchPrefix string) (map[string]string, error) {
	result := make(map[string]string)
	for _, entry := range os.Environ() {
		key, val, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		if matchPrefix != "" && !strings.HasPrefix(key, matchPrefix) {
			continue
		}
		relKey := strings.TrimPrefix(key, matchPrefix)
		if relKey == "" {
			continue
		}
		if len(val) > MaxValueSize {
			if f.logger != nil {
				f.logger.Warn("skipping %s: value size %d exceeds MaxValueSize", key, len(val))
			}
			continue
		}
		result[relKey] = val
	}
	return result, nil
}
