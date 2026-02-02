// Package resolver provides path-to-variable-name transformation and prefix handling.
package resolver

import (
	"strings"
)

// ApplyPrefix applies the prefix to the variable name based on the mode.
// In prepend mode, it adds the prefix to the variable name.
// In filter_only mode, it returns the variable name unchanged (filtering happens in fetcher).
// For invalid modes, returns the variable name unchanged to fail gracefully.
func ApplyPrefix(varName, prefix, mode string) string {
	// If no prefix configured, return unchanged
	if prefix == "" {
		return varName
	}

	switch mode {
	case "prepend":
		return PrependPrefix(varName, prefix)
	case "filter_only":
		// In filter_only mode, the varName should already contain the prefix from the path
		// Just return it unchanged - filtering happens in the fetcher
		return varName
	default:
		// For invalid modes, fail gracefully by returning unchanged
		return varName
	}
}

// PrependPrefix adds the prefix to the variable name.
func PrependPrefix(varName, prefix string) string {
	return prefix + varName
}

// FilterByPrefix checks if a variable name has the required prefix.
// Returns true if the variable should be accessible, false otherwise.
// If no prefix is configured (empty string), all variables are allowed.
func FilterByPrefix(varName, prefix string) bool {
	// If no prefix configured, allow all variables
	if prefix == "" {
		return true
	}

	// Check if variable name starts with the prefix (case-sensitive)
	return strings.HasPrefix(varName, prefix)
}
