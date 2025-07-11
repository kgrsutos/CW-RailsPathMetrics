package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Simple numeric ID",
			input: "/users/123",
			want:  "/users/:id",
		},
		{
			name:  "Multiple numeric IDs",
			input: "/posts/456/comments/789",
			want:  "/posts/:id/comments/:id",
		},
		{
			name:  "UUID format",
			input: "/api/v1/resources/550e8400-e29b-41d4-a716-446655440000",
			want:  "/api/v1/resources/:id",
		},
		{
			name:  "Mixed parameters",
			input: "/users/john/posts/123",
			want:  "/users/john/posts/:id",
		},
		{
			name:  "No parameters",
			input: "/about/us",
			want:  "/about/us",
		},
		{
			name:  "Root path",
			input: "/",
			want:  "/",
		},
		{
			name:  "Path with query string",
			input: "/users/123?page=1",
			want:  "/users/:id",
		},
		{
			name:  "Date format YYYY-MM-DD",
			input: "/posts/2023-01-15",
			want:  "/posts/:date",
		},
		{
			name:  "Path with extension",
			input: "/files/document.pdf",
			want:  "/files/document.pdf",
		},
		{
			name:  "API endpoint with version and ID",
			input: "/api/v2/orders/ORD-2023-001234",
			want:  "/api/v2/orders/:id",
		},
		{
			name:  "Complex nested path",
			input: "/organizations/123/teams/456/members/789",
			want:  "/organizations/:id/teams/:id/members/:id",
		},
		{
			name:  "Path with action",
			input: "/users/123/edit",
			want:  "/users/:id/edit",
		},
		{
			name:  "Hex ID",
			input: "/objects/a1b2c3d4e5f6",
			want:  "/objects/:id",
		},
		{
			name:  "Empty path segments",
			input: "/users//123",
			want:  "/users//:id",
		},
		{
			name:  "Trailing slash",
			input: "/users/123/",
			want:  "/users/:id/",
		},
		{
			name:  "Path with multiple query parameters",
			input: "/api/users/456?page=2&sort=name&filter=active",
			want:  "/api/users/:id",
		},
		{
			name:  "Path with empty query string",
			input: "/products/789?",
			want:  "/products/:id",
		},
		{
			name:  "Complex path with query parameters",
			input: "/organizations/123/teams/456?include=members&limit=50",
			want:  "/organizations/:id/teams/:id",
		},
		{
			name:  "Path with UUID and query parameters",
			input: "/items/550e8400-e29b-41d4-a716-446655440000?version=2",
			want:  "/items/:id",
		},
	}

	normalizer := NewNormalizer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.NormalizePath(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsNumericID(t *testing.T) {
	tests := []struct {
		name     string
		segment  string
		expected bool
	}{
		{"Pure numeric", "123", true},
		{"Large number", "999999999", true},
		{"Zero", "0", true},
		{"Alphabetic", "abc", false},
		{"Mixed alphanumeric", "abc123", false},
		{"Empty string", "", false},
		{"Special characters", "user@123", false},
		{"Decimal number", "123.45", false},
		{"Negative number", "-123", false},
	}

	normalizer := NewNormalizer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.isNumericID(tt.segment)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		name     string
		segment  string
		expected bool
	}{
		{
			name:     "Valid UUID v4",
			segment:  "550e8400-e29b-41d4-a716-446655440000",
			expected: true,
		},
		{
			name:     "Valid UUID v1",
			segment:  "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			expected: true,
		},
		{
			name:     "Uppercase UUID",
			segment:  "550E8400-E29B-41D4-A716-446655440000",
			expected: true,
		},
		{
			name:     "Invalid UUID - wrong format",
			segment:  "550e8400-e29b-41d4-a716",
			expected: false,
		},
		{
			name:     "Not a UUID",
			segment:  "not-a-uuid",
			expected: false,
		},
		{
			name:     "Numeric string",
			segment:  "123456789",
			expected: false,
		},
	}

	normalizer := NewNormalizer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.isUUID(tt.segment)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestIsHexID(t *testing.T) {
	tests := []struct {
		name     string
		segment  string
		expected bool
	}{
		{
			name:     "Valid hex ID",
			segment:  "a1b2c3d4e5f6",
			expected: true,
		},
		{
			name:     "Uppercase hex",
			segment:  "A1B2C3D4E5F6",
			expected: true,
		},
		{
			name:     "MongoDB ObjectId",
			segment:  "507f1f77bcf86cd799439011",
			expected: true,
		},
		{
			name:     "Short hex",
			segment:  "abc123",
			expected: true,
		},
		{
			name:     "Too short",
			segment:  "ab",
			expected: false,
		},
		{
			name:     "Contains non-hex",
			segment:  "xyz123",
			expected: false,
		},
		{
			name:     "Pure numeric",
			segment:  "123456",
			expected: false,
		},
	}

	normalizer := NewNormalizer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.isHexID(tt.segment)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestIsDateFormat(t *testing.T) {
	tests := []struct {
		name     string
		segment  string
		expected bool
	}{
		{
			name:     "Valid date YYYY-MM-DD",
			segment:  "2023-01-15",
			expected: true,
		},
		{
			name:     "Valid date YYYY-MM-DD",
			segment:  "2024-12-31",
			expected: true,
		},
		{
			name:     "Invalid date format",
			segment:  "2023/01/15",
			expected: false,
		},
		{
			name:     "Not a date",
			segment:  "not-a-date",
			expected: false,
		},
		{
			name:     "Partial date",
			segment:  "2023-01",
			expected: false,
		},
		{
			name:     "Invalid month",
			segment:  "2023-13-01",
			expected: false,
		},
		{
			name:     "Invalid day",
			segment:  "2023-02-30",
			expected: false,
		},
		{
			name:     "Valid leap year date",
			segment:  "2024-02-29",
			expected: true,
		},
		{
			name:     "Invalid leap year date",
			segment:  "2023-02-29",
			expected: false,
		},
	}

	normalizer := NewNormalizer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.isDateFormat(tt.segment)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestIsOrderID(t *testing.T) {
	tests := []struct {
		name     string
		segment  string
		expected bool
	}{
		{
			name:     "Order ID with ORD prefix",
			segment:  "ORD-2023-001234",
			expected: true,
		},
		{
			name:     "Invoice ID",
			segment:  "INV-2023-5678",
			expected: true,
		},
		{
			name:     "Reference ID",
			segment:  "REF-ABC-123",
			expected: true,
		},
		{
			name:     "User ID format",
			segment:  "USR-999999",
			expected: true,
		},
		{
			name:     "Simple numeric",
			segment:  "123456",
			expected: false,
		},
		{
			name:     "No dash",
			segment:  "ORD2023001234",
			expected: false,
		},
		{
			name:     "Single part",
			segment:  "ORDER",
			expected: false,
		},
	}

	normalizer := NewNormalizer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.isOrderID(tt.segment)
			assert.Equal(t, tt.expected, got)
		})
	}
}
