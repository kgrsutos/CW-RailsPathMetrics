package analyzer

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kgrsutos/cw-railspathmetrics/internal/models"
)

func TestAnalyzer_AnalyzeLogEvents(t *testing.T) {
	analyzer := NewAnalyzer()
	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC)

	tests := []struct {
		name      string
		logEvents []*models.LogEvent
		expected  *models.AnalysisResult
	}{
		{
			name: "analyze complete log events",
			logEvents: []*models.LogEvent{
				{
					ID:        "1",
					Message:   `Started GET "/users/123" for 127.0.0.1 at 2023-01-01 12:00:00 +0900 [abc123]`,
					Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				},
				{
					ID:        "2",
					Message:   `Completed 200 OK in 150ms (Views: 100.0ms | ActiveRecord: 50.0ms) [abc123]`,
					Timestamp: time.Date(2023, 1, 1, 12, 0, 1, 0, time.UTC),
				},
			},
			expected: &models.AnalysisResult{
				StartTime: startTime,
				EndTime:   endTime,
				TotalLogs: 2,
				PathMetrics: map[string]*models.PathMetrics{
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
		},
		{
			name:      "analyze empty log events",
			logEvents: []*models.LogEvent{},
			expected: &models.AnalysisResult{
				StartTime:   startTime,
				EndTime:     endTime,
				TotalLogs:   0,
				PathMetrics: map[string]*models.PathMetrics{},
			},
		},
		{
			name: "skip invalid log entries",
			logEvents: []*models.LogEvent{
				{
					ID:        "1",
					Message:   `Invalid log format`,
					Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				},
				{
					ID:        "2",
					Message:   `Started GET "/users/123" for 127.0.0.1 at 2023-01-01 12:00:00 +0900 [xyz789]`,
					Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				},
				{
					ID:        "3",
					Message:   `Completed 200 OK in 150ms [xyz789]`,
					Timestamp: time.Date(2023, 1, 1, 12, 0, 1, 0, time.UTC),
				},
			},
			expected: &models.AnalysisResult{
				StartTime: startTime,
				EndTime:   endTime,
				TotalLogs: 2,
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
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.AnalyzeLogEvents(tt.logEvents, startTime, endTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzer_OutputJSON(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name           string
		analysisResult *models.AnalysisResult
		expectedJSON   string
	}{
		{
			name: "output valid analysis result",
			analysisResult: &models.AnalysisResult{
				StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC),
				TotalLogs: 2,
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
				},
			},
			expectedJSON: `{
  "start_time": "2023-01-01T00:00:00Z",
  "end_time": "2023-01-01T23:59:59Z",
  "total_logs_analyzed": 2,
  "path_metrics": {
    "/users/:id": {
      "path": "/users/:id",
      "count": 1,
      "average_time_ms": 150,
      "min_time_ms": 150,
      "max_time_ms": 150,
      "status_codes": {
        "200": 1
      },
      "methods": {
        "GET": 1
      }
    }
  }
}`,
		},
		{
			name: "output empty analysis result",
			analysisResult: &models.AnalysisResult{
				StartTime:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				EndTime:     time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC),
				TotalLogs:   0,
				PathMetrics: map[string]*models.PathMetrics{},
			},
			expectedJSON: `{
  "start_time": "2023-01-01T00:00:00Z",
  "end_time": "2023-01-01T23:59:59Z",
  "total_logs_analyzed": 0,
  "path_metrics": {}
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := analyzer.OutputJSON(tt.analysisResult, &buf)
			require.NoError(t, err)

			// Parse both expected and actual JSON to compare structure
			var expectedData, actualData interface{}
			err = json.Unmarshal([]byte(tt.expectedJSON), &expectedData)
			require.NoError(t, err)
			err = json.Unmarshal(buf.Bytes(), &actualData)
			require.NoError(t, err)

			assert.Equal(t, expectedData, actualData)
		})
	}
}

func TestNewAnalyzer(t *testing.T) {
	analyzer := NewAnalyzer()
	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.parser)
	assert.NotNil(t, analyzer.normalizer)
	assert.NotNil(t, analyzer.aggregator)
}
