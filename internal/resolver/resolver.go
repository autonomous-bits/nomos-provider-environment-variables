package resolver

import (
	"errors"
	"strings"
)

var (
	// ErrEmptyPath is returned when an empty path is provided
	ErrEmptyPath = errors.New("path cannot be empty")
	// ErrEmptySegment is returned when a path contains an empty segment
	ErrEmptySegment = errors.New("path segments cannot be empty")
)

// Resolver transforms hierarchical paths into environment variable names
// using configurable separator, case conversion, and prefix handling.
type Resolver struct {
	separator     string
	caseTransform string
	prefix        string
	prefixMode    string
}

// NewResolver creates a new Resolver with the specified configuration.
// The separator is used to join path segments, caseTransform specifies how to
// transform the case of each segment ("upper", "lower", or "preserve"),
// prefix is the prefix to apply, and prefixMode controls prefix behavior
// ("prepend" or "filter_only").
func NewResolver(separator, caseTransform, prefix, prefixMode string) *Resolver {
	return &Resolver{
		separator:     separator,
		caseTransform: caseTransform,
		prefix:        prefix,
		prefixMode:    prefixMode,
	}
}

// Transform converts a hierarchical path into an environment variable name.
// It validates the path, applies case transformation to each segment,
// joins them with the configured separator, and applies prefix based on mode.
//
// Example: []string{"database", "host"} with separator="_", transform="upper",
// prefix="MYAPP_", and mode="prepend" returns "MYAPP_DATABASE_HOST".
//
// Returns an error if the path is empty, contains empty segments, or
// prefix mode is invalid.
func (r *Resolver) Transform(path []string) (string, error) {
	// Validate path is not empty
	if len(path) == 0 {
		return "", ErrEmptyPath
	}

	// Validate no segments are empty or only whitespace
	for i, segment := range path {
		if strings.TrimSpace(segment) == "" {
			return "", ErrEmptySegment
		}
		// Store the trimmed version to avoid issues
		path[i] = segment
	}

	// Transform all segments
	transformed := TransformSegments(path, r.caseTransform)

	// Join with separator
	transformedName := strings.Join(transformed, r.separator)

	// Apply prefix based on mode
	varName := ApplyPrefix(transformedName, r.prefix, r.prefixMode)

	return varName, nil
}

// BuildPrefix computes the environment variable prefix used for wildcard matching.
// namespacePath contains all path segments preceding the terminal "*".
//
// Root wildcard (empty namespacePath):
//   - Returns r.prefix when prefixMode == "prepend" and prefix is non-empty.
//   - Otherwise returns "".
//
// Non-root wildcard (non-empty namespacePath):
//   - Validates that no segment is empty.
//   - Applies case transformation to each segment.
//   - Joins segments with the configured separator and appends a trailing separator
//     to enforce full-segment boundary matching.
//   - Prepends r.prefix when prefixMode == "prepend" and prefix is non-empty.
func (r *Resolver) BuildPrefix(namespacePath []string) (string, error) {
	if len(namespacePath) == 0 {
		if r.prefixMode == "prepend" && r.prefix != "" {
			return r.prefix, nil
		}
		return "", nil
	}

	for _, seg := range namespacePath {
		if strings.TrimSpace(seg) == "" {
			return "", ErrEmptySegment
		}
	}

	transformed := TransformSegments(namespacePath, r.caseTransform)
	joined := strings.Join(transformed, r.separator)
	base := joined + r.separator

	if r.prefixMode == "prepend" && r.prefix != "" {
		return r.prefix + base, nil
	}
	return base, nil
}
