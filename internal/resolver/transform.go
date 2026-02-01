package resolver

import "strings"

// ToUpperCase converts a string to uppercase using Unicode case mapping.
func ToUpperCase(s string) string {
	return strings.ToUpper(s)
}

// ToLowerCase converts a string to lowercase using Unicode case mapping.
func ToLowerCase(s string) string {
	return strings.ToLower(s)
}

// PreserveCase returns the string unchanged, preserving its original case.
func PreserveCase(s string) string {
	return s
}

// TransformSegment applies the specified case transformation to a single path segment.
// Valid transformations are "upper", "lower", and "preserve".
func TransformSegment(segment, caseTransform string) string {
	switch caseTransform {
	case "upper":
		return ToUpperCase(segment)
	case "lower":
		return ToLowerCase(segment)
	case "preserve":
		return PreserveCase(segment)
	default:
		// Default to preserve if unknown transformation
		return segment
	}
}

// TransformSegments applies the specified case transformation to all path segments.
// Returns a new slice with transformed segments.
func TransformSegments(segments []string, caseTransform string) []string {
	if len(segments) == 0 {
		return []string{}
	}

	transformed := make([]string, len(segments))
	for i, segment := range segments {
		transformed[i] = TransformSegment(segment, caseTransform)
	}
	return transformed
}
