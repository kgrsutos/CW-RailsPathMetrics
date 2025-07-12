package analyzer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kgrsutos/cw-railspathmetrics/internal/models"
)

func TestAggregator_MatchRequestPairs(t *testing.T) {
	aggregator := NewAggregator()

	tests := []struct {
		name     string
		entries  []*models.LogEntry
		expected []*models.RequestPair
	}{
		{
			name: "match single started and completed logs with same session ID",
			entries: []*models.LogEntry{
				{
					Type:      "Started",
					Method:    "GET",
					Path:      "/users/123",
					Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					SessionID: "abc123",
				},
				{
					Type:       "Completed",
					StatusCode: 200,
					StatusText: "OK",
					Duration:   150,
					SessionID:  "abc123",
				},
			},
			expected: []*models.RequestPair{
				{
					Started: &models.LogEntry{
						Type:      "Started",
						Method:    "GET",
						Path:      "/users/123",
						Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
						SessionID: "abc123",
					},
					Completed: &models.LogEntry{
						Type:       "Completed",
						StatusCode: 200,
						StatusText: "OK",
						Duration:   150,
						SessionID:  "abc123",
					},
				},
			},
		},
		{
			name: "match multiple started and completed logs",
			entries: []*models.LogEntry{
				{
					Type:      "Started",
					Method:    "GET",
					Path:      "/users/123",
					Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					SessionID: "abc123",
				},
				{
					Type:      "Started",
					Method:    "POST",
					Path:      "/posts",
					Timestamp: time.Date(2023, 1, 1, 12, 1, 0, 0, time.UTC),
					SessionID: "def456",
				},
				{
					Type:       "Completed",
					StatusCode: 200,
					StatusText: "OK",
					Duration:   150,
					SessionID:  "abc123",
				},
				{
					Type:       "Completed",
					StatusCode: 201,
					StatusText: "Created",
					Duration:   250,
					SessionID:  "def456",
				},
			},
			expected: []*models.RequestPair{
				{
					Started: &models.LogEntry{
						Type:      "Started",
						Method:    "GET",
						Path:      "/users/123",
						Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
						SessionID: "abc123",
					},
					Completed: &models.LogEntry{
						Type:       "Completed",
						StatusCode: 200,
						StatusText: "OK",
						Duration:   150,
						SessionID:  "abc123",
					},
				},
				{
					Started: &models.LogEntry{
						Type:      "Started",
						Method:    "POST",
						Path:      "/posts",
						Timestamp: time.Date(2023, 1, 1, 12, 1, 0, 0, time.UTC),
						SessionID: "def456",
					},
					Completed: &models.LogEntry{
						Type:       "Completed",
						StatusCode: 201,
						StatusText: "Created",
						Duration:   250,
						SessionID:  "def456",
					},
				},
			},
		},
		{
			name: "handle orphaned started log (no matching completed)",
			entries: []*models.LogEntry{
				{
					Type:      "Started",
					Method:    "GET",
					Path:      "/users/123",
					Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					SessionID: "orphan123",
				},
			},
			expected: []*models.RequestPair{},
		},
		{
			name: "handle orphaned completed log (no matching started)",
			entries: []*models.LogEntry{
				{
					Type:       "Completed",
					StatusCode: 200,
					StatusText: "OK",
					Duration:   150,
					SessionID:  "abc123",
				},
			},
			expected: []*models.RequestPair{},
		},
		{
			name: "ignore logs with mismatched session IDs",
			entries: []*models.LogEntry{
				{
					Type:      "Started",
					Method:    "GET",
					Path:      "/users/123",
					Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					SessionID: "session1",
				},
				{
					Type:       "Completed",
					StatusCode: 200,
					StatusText: "OK",
					Duration:   150,
					SessionID:  "session2",
				},
			},
			expected: []*models.RequestPair{},
		},
		{
			name: "ignore logs without session ID",
			entries: []*models.LogEntry{
				{
					Type:      "Started",
					Method:    "GET",
					Path:      "/users/123",
					Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					SessionID: "",
				},
				{
					Type:       "Completed",
					StatusCode: 200,
					StatusText: "OK",
					Duration:   150,
					SessionID:  "",
				},
			},
			expected: []*models.RequestPair{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregator.MatchRequestPairs(tt.entries)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAggregator_AggregateMetrics(t *testing.T) {
	aggregator := NewAggregator()
	normalizer := NewNormalizer()

	tests := []struct {
		name     string
		pairs    []*models.RequestPair
		expected map[string]*models.PathMetrics
	}{
		{
			name: "aggregate single request pair",
			pairs: []*models.RequestPair{
				{
					Started: &models.LogEntry{
						Type:   "Started",
						Method: "GET",
						Path:   "/users/123",
					},
					Completed: &models.LogEntry{
						Type:         "Completed",
						StatusCode:   200,
						StatusText:   "OK",
						Duration:     150,
						ViewDuration: 100.0,
						DBDuration:   50.0,
					},
				},
			},
			expected: map[string]*models.PathMetrics{
				"/users/:id": {
					Path:              "/users/:id",
					Count:             1,
					AverageTime:       150.0,
					MinTime:           150,
					MaxTime:           150,
					StatusCodes:       map[int]int{200: 1},
					Methods:           map[string]int{"GET": 1},
					TotalViewDuration: 100.0,
					TotalDBDuration:   50.0,
				},
			},
		},
		{
			name: "aggregate multiple request pairs for same path",
			pairs: []*models.RequestPair{
				{
					Started: &models.LogEntry{
						Type:   "Started",
						Method: "GET",
						Path:   "/users/123",
					},
					Completed: &models.LogEntry{
						Type:         "Completed",
						StatusCode:   200,
						StatusText:   "OK",
						Duration:     150,
						ViewDuration: 100.0,
						DBDuration:   50.0,
					},
				},
				{
					Started: &models.LogEntry{
						Type:   "Started",
						Method: "GET",
						Path:   "/users/456",
					},
					Completed: &models.LogEntry{
						Type:         "Completed",
						StatusCode:   200,
						StatusText:   "OK",
						Duration:     250,
						ViewDuration: 200.0,
						DBDuration:   50.0,
					},
				},
			},
			expected: map[string]*models.PathMetrics{
				"/users/:id": {
					Path:              "/users/:id",
					Count:             2,
					AverageTime:       200.0,
					MinTime:           150,
					MaxTime:           250,
					StatusCodes:       map[int]int{200: 2},
					Methods:           map[string]int{"GET": 2},
					TotalViewDuration: 300.0,
					TotalDBDuration:   100.0,
				},
			},
		},
		{
			name: "aggregate different paths",
			pairs: []*models.RequestPair{
				{
					Started: &models.LogEntry{
						Type:   "Started",
						Method: "GET",
						Path:   "/users/123",
					},
					Completed: &models.LogEntry{
						Type:       "Completed",
						StatusCode: 200,
						StatusText: "OK",
						Duration:   150,
					},
				},
				{
					Started: &models.LogEntry{
						Type:   "Started",
						Method: "POST",
						Path:   "/posts",
					},
					Completed: &models.LogEntry{
						Type:       "Completed",
						StatusCode: 201,
						StatusText: "Created",
						Duration:   250,
					},
				},
			},
			expected: map[string]*models.PathMetrics{
				"/users/:id": {
					Path:        "/users/:id",
					Count:       1,
					AverageTime: 150.0,
					MinTime:     150,
					MaxTime:     150,
					StatusCodes: map[int]int{200: 1},
					Methods:     map[string]int{"GET": 1},
				},
				"/posts": {
					Path:        "/posts",
					Count:       1,
					AverageTime: 250.0,
					MinTime:     250,
					MaxTime:     250,
					StatusCodes: map[int]int{201: 1},
					Methods:     map[string]int{"POST": 1},
				},
			},
		},
		{
			name: "aggregate mixed methods and status codes",
			pairs: []*models.RequestPair{
				{
					Started: &models.LogEntry{
						Type:   "Started",
						Method: "GET",
						Path:   "/users/123",
					},
					Completed: &models.LogEntry{
						Type:       "Completed",
						StatusCode: 200,
						StatusText: "OK",
						Duration:   150,
					},
				},
				{
					Started: &models.LogEntry{
						Type:   "Started",
						Method: "POST",
						Path:   "/users/456",
					},
					Completed: &models.LogEntry{
						Type:       "Completed",
						StatusCode: 404,
						StatusText: "Not Found",
						Duration:   100,
					},
				},
			},
			expected: map[string]*models.PathMetrics{
				"/users/:id": {
					Path:        "/users/:id",
					Count:       2,
					AverageTime: 125.0,
					MinTime:     100,
					MaxTime:     150,
					StatusCodes: map[int]int{200: 1, 404: 1},
					Methods:     map[string]int{"GET": 1, "POST": 1},
				},
			},
		},
		{
			name: "handle zero duration logs",
			pairs: []*models.RequestPair{
				{
					Started: &models.LogEntry{
						Type:      "Started",
						Method:    "GET",
						Path:      "/health",
						Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
						SessionID: "health123",
					},
					Completed: &models.LogEntry{
						Type:         "Completed",
						StatusCode:   200,
						StatusText:   "OK",
						Duration:     0, // Zero duration
						ViewDuration: 0,
						DBDuration:   0,
						SessionID:    "health123",
					},
				},
			},
			expected: map[string]*models.PathMetrics{
				"/health": {
					Path:              "/health",
					Count:             1,
					AverageTime:       0,
					MinTime:           0,
					MaxTime:           0,
					StatusCodes:       map[int]int{200: 1},
					Methods:           map[string]int{"GET": 1},
					TotalViewDuration: 0,
					TotalDBDuration:   0,
				},
			},
		},
		{
			name:     "handle empty request pairs",
			pairs:    []*models.RequestPair{},
			expected: map[string]*models.PathMetrics{},
		},
		{
			name: "exclude paths based on path exclusion rules",
			pairs: []*models.RequestPair{
				{
					Started: &models.LogEntry{
						Type:      "Started",
						Method:    "GET",
						Path:      "/users/123",
						Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
						SessionID: "user123",
					},
					Completed: &models.LogEntry{
						Type:       "Completed",
						StatusCode: 200,
						StatusText: "OK",
						Duration:   150,
						SessionID:  "user123",
					},
				},
				{
					Started: &models.LogEntry{
						Type:      "Started",
						Method:    "GET",
						Path:      "/rails/active_storage/blobs/456",
						Timestamp: time.Date(2023, 1, 1, 12, 1, 0, 0, time.UTC),
						SessionID: "storage456",
					},
					Completed: &models.LogEntry{
						Type:       "Completed",
						StatusCode: 200,
						StatusText: "OK",
						Duration:   100,
						SessionID:  "storage456",
					},
				},
			},
			expected: map[string]*models.PathMetrics{
				"/users/:id": {
					Path:        "/users/:id",
					Count:       1,
					AverageTime: 150.0,
					MinTime:     150,
					MaxTime:     150,
					StatusCodes: map[int]int{200: 1},
					Methods:     map[string]int{"GET": 1},
				},
				// Note: /rails/active_storage path should be excluded
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregator.AggregateMetrics(tt.pairs, normalizer)
			assert.Equal(t, len(tt.expected), len(result))
			for path, expectedMetrics := range tt.expected {
				actualMetrics, exists := result[path]
				require.True(t, exists, "Expected path %s not found in result", path)
				assert.Equal(t, expectedMetrics, actualMetrics)
			}
		})
	}
}

func TestAggregator_AnalyzeLogs(t *testing.T) {
	aggregator := NewAggregator()
	normalizer := NewNormalizer()
	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC)

	tests := []struct {
		name     string
		entries  []*models.LogEntry
		expected *models.AnalysisResult
	}{
		{
			name: "analyze complete log entries",
			entries: []*models.LogEntry{
				{
					Type:      "Started",
					Method:    "GET",
					Path:      "/users/123",
					Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
					SessionID: "abc123",
				},
				{
					Type:       "Completed",
					StatusCode: 200,
					StatusText: "OK",
					Duration:   150,
					SessionID:  "abc123",
				},
				{
					Type:      "Started",
					Method:    "POST",
					Path:      "/posts",
					Timestamp: time.Date(2023, 1, 1, 12, 1, 0, 0, time.UTC),
					SessionID: "def456",
				},
				{
					Type:       "Completed",
					StatusCode: 201,
					StatusText: "Created",
					Duration:   250,
					SessionID:  "def456",
				},
			},
			expected: &models.AnalysisResult{
				StartTime: startTime,
				EndTime:   endTime,
				TotalLogs: 4,
				PathMetrics: map[string]*models.PathMetrics{
					"/users/:id": {
						Path:        "/users/:id",
						Count:       1,
						AverageTime: 150.0,
						MinTime:     150,
						MaxTime:     150,
						StatusCodes: map[int]int{200: 1},
						Methods:     map[string]int{"GET": 1},
					},
					"/posts": {
						Path:        "/posts",
						Count:       1,
						AverageTime: 250.0,
						MinTime:     250,
						MaxTime:     250,
						StatusCodes: map[int]int{201: 1},
						Methods:     map[string]int{"POST": 1},
					},
				},
			},
		},
		{
			name:    "analyze empty log entries",
			entries: []*models.LogEntry{},
			expected: &models.AnalysisResult{
				StartTime:   startTime,
				EndTime:     endTime,
				TotalLogs:   0,
				PathMetrics: map[string]*models.PathMetrics{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aggregator.AnalyzeLogs(tt.entries, normalizer, startTime, endTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewAggregator(t *testing.T) {
	aggregator := NewAggregator()
	assert.NotNil(t, aggregator)
}
