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
			expectedJSON: `[
    {
        "path": "/users/:id",
        "count": 1,
        "max_time_ms": 150,
        "min_time_ms": 150,
        "avg_time_ms": "150"
    }
]`,
		},
		{
			name: "output analysis result with view and DB durations",
			analysisResult: &models.AnalysisResult{
				StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC),
				TotalLogs: 4,
				PathMetrics: map[string]*models.PathMetrics{
					"/api/posts/:id": {
						Path:              "/api/posts/:id",
						Count:             2,
						AverageTime:       200.0,
						MinTime:           150,
						MaxTime:           250,
						StatusCodes:       map[int]int{200: 2},
						Methods:           map[string]int{"GET": 2},
						TotalViewDuration: 180.5,
						TotalDBDuration:   95.2,
					},
				},
			},
			expectedJSON: `[
    {
        "path": "/api/posts/:id",
        "count": 2,
        "max_time_ms": 250,
        "min_time_ms": 150,
        "avg_time_ms": "200"
    }
]`,
		},
		{
			name: "output empty analysis result",
			analysisResult: &models.AnalysisResult{
				StartTime:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				EndTime:     time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC),
				TotalLogs:   0,
				PathMetrics: map[string]*models.PathMetrics{},
			},
			expectedJSON: `[]`,
		},
		{
			name: "output multiple paths",
			analysisResult: &models.AnalysisResult{
				StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC),
				TotalLogs: 6,
				PathMetrics: map[string]*models.PathMetrics{
					"/path1/path2": {
						Path:        "/path1/path2",
						Count:       100,
						AverageTime: 1000.0,
						MinTime:     640,
						MaxTime:     2300,
						StatusCodes: map[int]int{200: 100},
						Methods:     map[string]int{"GET": 100},
					},
					"/path1/path3": {
						Path:        "/path1/path3",
						Count:       50,
						AverageTime: 1200.0,
						MinTime:     840,
						MaxTime:     2200,
						StatusCodes: map[int]int{200: 50},
						Methods:     map[string]int{"POST": 50},
					},
				},
			},
			expectedJSON: `[
    {
        "path": "/path1/path2",
        "count": 100,
        "max_time_ms": 2300,
        "min_time_ms": 640,
        "avg_time_ms": "1000"
    },
    {
        "path": "/path1/path3",
        "count": 50,
        "max_time_ms": 2200,
        "min_time_ms": 840,
        "avg_time_ms": "1200"
    }
]`,
		},
		{
			name: "output analysis result with zero durations",
			analysisResult: &models.AnalysisResult{
				StartTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2023, 1, 1, 23, 59, 59, 0, time.UTC),
				TotalLogs: 2,
				PathMetrics: map[string]*models.PathMetrics{
					"/health": {
						Path:              "/health",
						Count:             5,
						AverageTime:       0,
						MinTime:           0,
						MaxTime:           0,
						StatusCodes:       map[int]int{200: 5},
						Methods:           map[string]int{"GET": 5},
						TotalViewDuration: 0,
						TotalDBDuration:   0,
					},
				},
			},
			expectedJSON: `[
    {
        "path": "/health",
        "count": 5,
        "max_time_ms": 0,
        "min_time_ms": 0,
        "avg_time_ms": "0"
    }
]`,
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

func TestNewAnalyzerWithConfig(t *testing.T) {
	tests := []struct {
		name          string
		configPath    string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid config path",
			configPath:  "config/excluded_paths.yml",
			expectError: false,
		},
		{
			name:          "invalid config path",
			configPath:    "non/existent/path.yml",
			expectError:   true,
			errorContains: "failed to create aggregator with config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, err := NewAnalyzerWithConfig(tt.configPath)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, analyzer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, analyzer)
				assert.NotNil(t, analyzer.parser)
				assert.NotNil(t, analyzer.normalizer)
				assert.NotNil(t, analyzer.aggregator)
			}
		})
	}
}
