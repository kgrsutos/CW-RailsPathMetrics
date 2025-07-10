package analyzer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kgrsutos/cw-railspathmetrics/internal/models"
)

// mustParseTime is a helper function to parse time in tests
func mustParseTime(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05 -0700", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseLogEntry(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *models.LogEntry
		wantErr bool
	}{
		{
			name:  "Started log entry",
			input: `Started GET "/users/123" for 127.0.0.1 at 2023-01-01 12:00:00 +0900`,
			want: &models.LogEntry{
				Type:      "Started",
				Method:    "GET",
				Path:      "/users/123",
				Timestamp: mustParseTime("2023-01-01 12:00:00 +0900"),
			},
			wantErr: false,
		},
		{
			name:  "Completed log entry with session ID",
			input: `Completed 200 OK in 150ms (Views: 100.0ms | ActiveRecord: 50.0ms) [a1b2c3d4]`,
			want: &models.LogEntry{
				Type:         "Completed",
				StatusCode:   200,
				StatusText:   "OK",
				Duration:     150,
				ViewDuration: 100.0,
				DBDuration:   50.0,
				SessionID:    "a1b2c3d4",
			},
			wantErr: false,
		},
		{
			name:  "Completed log entry without session ID",
			input: `Completed 404 Not Found in 50ms`,
			want: &models.LogEntry{
				Type:       "Completed",
				StatusCode: 404,
				StatusText: "Not Found",
				Duration:   50,
			},
			wantErr: false,
		},
		{
			name:  "Started log with query parameters",
			input: `Started POST "/api/users?page=1&limit=10" for 192.168.1.1 at 2023-01-01 15:30:45 +0900`,
			want: &models.LogEntry{
				Type:      "Started",
				Method:    "POST",
				Path:      "/api/users?page=1&limit=10",
				Timestamp: mustParseTime("2023-01-01 15:30:45 +0900"),
			},
			wantErr: false,
		},
		{
			name:  "Started log with nested path",
			input: `Started DELETE "/posts/456/comments/789" for 10.0.0.1 at 2023-02-15 09:15:30 +0900`,
			want: &models.LogEntry{
				Type:      "Started",
				Method:    "DELETE",
				Path:      "/posts/456/comments/789",
				Timestamp: mustParseTime("2023-02-15 09:15:30 +0900"),
			},
			wantErr: false,
		},
		{
			name:    "Invalid log format",
			input:   `Random log entry that doesn't match Rails format`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Empty log entry",
			input:   "",
			want:    nil,
			wantErr: true,
		},
		{
			name:  "Completed log with redirect status",
			input: `Completed 302 Found in 25ms`,
			want: &models.LogEntry{
				Type:       "Completed",
				StatusCode: 302,
				StatusText: "Found",
				Duration:   25,
			},
			wantErr: false,
		},
		{
			name:  "Completed log with server error",
			input: `Completed 500 Internal Server Error in 1000ms`,
			want: &models.LogEntry{
				Type:       "Completed",
				StatusCode: 500,
				StatusText: "Internal Server Error",
				Duration:   1000,
			},
			wantErr: false,
		},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.ParseLogEntry(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Session ID in brackets",
			input: `Completed 200 OK in 150ms [abc123]`,
			want:  "abc123",
		},
		{
			name:  "No session ID",
			input: `Completed 200 OK in 150ms`,
			want:  "",
		},
		{
			name:  "Session ID with special characters",
			input: `Completed 200 OK in 150ms [a1b2-c3d4_e5f6]`,
			want:  "a1b2-c3d4_e5f6",
		},
		{
			name:  "Multiple brackets - takes last one",
			input: `Completed 200 OK in 150ms (Views: [test]) [session123]`,
			want:  "session123",
		},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.extractSessionID(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsStartedLog(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "Valid Started log",
			input: `Started GET "/users" for 127.0.0.1 at 2023-01-01 12:00:00 +0900`,
			want:  true,
		},
		{
			name:  "Completed log",
			input: `Completed 200 OK in 150ms`,
			want:  false,
		},
		{
			name:  "Random text starting with Started",
			input: `Started processing but not a Rails log`,
			want:  false,
		},
		{
			name:  "Empty string",
			input: "",
			want:  false,
		},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.isStartedLog(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsCompletedLog(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "Valid Completed log with details",
			input: `Completed 200 OK in 150ms (Views: 100.0ms | ActiveRecord: 50.0ms)`,
			want:  true,
		},
		{
			name:  "Simple Completed log",
			input: `Completed 404 Not Found in 50ms`,
			want:  true,
		},
		{
			name:  "Started log",
			input: `Started GET "/users" for 127.0.0.1 at 2023-01-01 12:00:00 +0900`,
			want:  false,
		},
		{
			name:  "Random text starting with Completed",
			input: `Completed task but not a Rails log`,
			want:  false,
		},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.isCompletedLog(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
