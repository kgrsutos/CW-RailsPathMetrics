package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
		hasError bool
	}{
		{
			name:     "valid JST time",
			input:    "2023-01-01T12:00:00",
			expected: time.Date(2023, 1, 1, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
			hasError: false,
		},
		{
			name:     "invalid format",
			input:    "2023-01-01 12:00:00",
			hasError: true,
		},
		{
			name:     "invalid date",
			input:    "2023-13-01T12:00:00",
			hasError: true,
		},
	}

	jst, err := time.LoadLocation("Asia/Tokyo")
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := time.ParseInLocation("2006-01-02T15:04:05", tt.input, jst)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.UTC(), parsed.UTC())
			}
		})
	}
}

func TestAnalyzeCommand(t *testing.T) {
	// Test that the analyze command is properly registered
	assert.NotNil(t, analyzeCmd)
	assert.Equal(t, "analyze", analyzeCmd.Use)
	assert.Equal(t, "Analyze CloudWatch logs for Rails request metrics", analyzeCmd.Short)

	// Test that flags exist
	assert.NotNil(t, analyzeCmd.Flags().Lookup("start"))
	assert.NotNil(t, analyzeCmd.Flags().Lookup("end"))
	assert.NotNil(t, analyzeCmd.Flags().Lookup("log-group"))
	assert.NotNil(t, analyzeCmd.Flags().Lookup("profile"))
	assert.NotNil(t, analyzeCmd.Flags().Lookup("config"))
}
