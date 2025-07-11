package config

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathExcluder_ShouldExclude(t *testing.T) {
	tests := []struct {
		name     string
		rules    []ExclusionRule
		path     string
		expected bool
	}{
		{
			name: "exact match exclusion",
			rules: []ExclusionRule{
				{Exact: "/health"},
			},
			path:     "/health",
			expected: true,
		},
		{
			name: "exact match no exclusion",
			rules: []ExclusionRule{
				{Exact: "/health"},
			},
			path:     "/users",
			expected: false,
		},
		{
			name: "prefix match exclusion",
			rules: []ExclusionRule{
				{Prefix: "/rails/active_storage"},
			},
			path:     "/rails/active_storage/blobs/123",
			expected: true,
		},
		{
			name: "prefix match no exclusion",
			rules: []ExclusionRule{
				{Prefix: "/rails/active_storage"},
			},
			path:     "/rails/application",
			expected: false,
		},
		{
			name: "pattern match exclusion",
			rules: []ExclusionRule{
				{Pattern: "^/api/v[0-9]+/.*"},
			},
			path:     "/api/v1/users",
			expected: true,
		},
		{
			name: "pattern match no exclusion",
			rules: []ExclusionRule{
				{Pattern: "^/api/v[0-9]+/.*"},
			},
			path:     "/api/users",
			expected: false,
		},
		{
			name: "multiple rules - first matches",
			rules: []ExclusionRule{
				{Exact: "/health"},
				{Prefix: "/assets"},
			},
			path:     "/health",
			expected: true,
		},
		{
			name: "multiple rules - second matches",
			rules: []ExclusionRule{
				{Exact: "/health"},
				{Prefix: "/assets"},
			},
			path:     "/assets/css/style.css",
			expected: true,
		},
		{
			name: "multiple rules - no match",
			rules: []ExclusionRule{
				{Exact: "/health"},
				{Prefix: "/assets"},
			},
			path:     "/users/123",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluder := &PathExcluder{
				config: &ExclusionConfig{
					ExcludedPaths: tt.rules,
				},
				compiledRegexs: make([]*regexp.Regexp, len(tt.rules)),
			}

			// Compile regex patterns
			for i, rule := range tt.rules {
				if rule.Pattern != "" {
					regex, err := regexp.Compile(rule.Pattern)
					require.NoError(t, err)
					excluder.compiledRegexs[i] = regex
				}
			}

			result := excluder.ShouldExclude(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewDefaultPathExcluder(t *testing.T) {
	excluder := NewDefaultPathExcluder()
	assert.NotNil(t, excluder)

	// Test default exclusion: /rails/active_storage
	assert.True(t, excluder.ShouldExclude("/rails/active_storage/blobs/123"))
	assert.True(t, excluder.ShouldExclude("/rails/active_storage/representations/456"))
	assert.False(t, excluder.ShouldExclude("/rails/application"))
	assert.False(t, excluder.ShouldExclude("/users/123"))
}

func TestNewPathExcluder_WithConfigFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_exclusions.yml")

	configContent := `excluded_paths:
  - exact: "/health"
  - prefix: "/assets"
  - pattern: "^/api/internal/.*"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	excluder, err := NewPathExcluder(configPath)
	require.NoError(t, err)
	assert.NotNil(t, excluder)

	// Test exclusions from config file
	assert.True(t, excluder.ShouldExclude("/health"))
	assert.True(t, excluder.ShouldExclude("/assets/css/style.css"))
	assert.True(t, excluder.ShouldExclude("/api/internal/metrics"))
	assert.False(t, excluder.ShouldExclude("/users/123"))
}

func TestNewPathExcluder_InvalidConfigFile(t *testing.T) {
	// Test with non-existent file
	_, err := NewPathExcluder("/non/existent/file.yml")
	assert.Error(t, err)

	// Test with invalid YAML
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yml")

	invalidContent := `excluded_paths:
  - exact: "/health"
    invalid_yaml: [
`

	err = os.WriteFile(configPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	_, err = NewPathExcluder(configPath)
	assert.Error(t, err)
}

func TestNewPathExcluder_InvalidRegexPattern(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid_regex.yml")

	configContent := `excluded_paths:
  - pattern: "[invalid_regex"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	_, err = NewPathExcluder(configPath)
	assert.Error(t, err)
}
