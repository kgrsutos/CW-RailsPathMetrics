package analyzer

import (
	"regexp"
	"strings"
)

var (
	// Regular expressions for ID detection
	uuidRegex  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	hexIDRegex = regexp.MustCompile(`^[0-9a-fA-F]{6,}$`)
	dateRegex  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	orderIDRegex = regexp.MustCompile(`^[A-Z]{3,}-[A-Z0-9]+-[0-9]+$|^[A-Z]{3,}-[0-9]+$`)
)

// Normalizer handles path normalization
type Normalizer struct{}

// NewNormalizer creates a new Normalizer instance
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// NormalizePath normalizes a request path by replacing dynamic segments with placeholders
func (n *Normalizer) NormalizePath(path string) string {
	// Split path and query string
	parts := strings.SplitN(path, "?", 2)
	pathPart := parts[0]
	queryPart := ""
	if len(parts) > 1 {
		queryPart = "?" + parts[1]
	}

	// Split path into segments
	segments := strings.Split(pathPart, "/")
	
	// Process each segment
	for i, segment := range segments {
		if segment == "" {
			continue
		}

		// Check if segment should be replaced
		if n.shouldNormalize(segment) {
			segments[i] = n.getPlaceholder(segment)
		}
	}

	// Reconstruct path
	normalizedPath := strings.Join(segments, "/")
	return normalizedPath + queryPart
}

// shouldNormalize determines if a path segment should be normalized
func (n *Normalizer) shouldNormalize(segment string) bool {
	return n.isNumericID(segment) ||
		n.isUUID(segment) ||
		n.isHexID(segment) ||
		n.isDateFormat(segment) ||
		n.isOrderID(segment)
}

// getPlaceholder returns the appropriate placeholder for a segment
func (n *Normalizer) getPlaceholder(segment string) string {
	if n.isDateFormat(segment) {
		return ":date"
	}
	return ":id"
}

// isNumericID checks if a segment is a numeric ID
func (n *Normalizer) isNumericID(segment string) bool {
	if segment == "" {
		return false
	}
	
	for _, ch := range segment {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// isUUID checks if a segment is a UUID
func (n *Normalizer) isUUID(segment string) bool {
	return uuidRegex.MatchString(segment)
}

// isHexID checks if a segment is a hexadecimal ID (at least 6 characters)
func (n *Normalizer) isHexID(segment string) bool {
	if len(segment) < 6 {
		return false
	}
	
	// Don't match pure numeric strings
	hasLetter := false
	for _, ch := range segment {
		if (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			hasLetter = true
			break
		}
	}
	
	if !hasLetter {
		return false
	}
	
	return hexIDRegex.MatchString(segment)
}

// isDateFormat checks if a segment is in date format (YYYY-MM-DD)
func (n *Normalizer) isDateFormat(segment string) bool {
	if !dateRegex.MatchString(segment) {
		return false
	}
	
	// Basic validation of date components
	parts := strings.Split(segment, "-")
	month := parts[1]
	
	// Check month is valid (01-12)
	if month < "01" || month > "12" {
		return false
	}
	
	return true
}

// isOrderID checks if a segment looks like an order/reference ID
func (n *Normalizer) isOrderID(segment string) bool {
	return orderIDRegex.MatchString(segment)
}