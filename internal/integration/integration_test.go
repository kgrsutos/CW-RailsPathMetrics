package integration

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kgrsutos/cw-railspathmetrics/internal/analyzer"
	"github.com/kgrsutos/cw-railspathmetrics/internal/cloudwatch"
	"github.com/kgrsutos/cw-railspathmetrics/internal/models"
)

// MockCloudWatchLogsAPI implements cloudwatch.CloudWatchLogsAPI for testing
type MockCloudWatchLogsAPI struct {
	mock.Mock
}

func (m *MockCloudWatchLogsAPI) FilterLogEvents(ctx context.Context, params *cloudwatchlogs.FilterLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.FilterLogEventsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudwatchlogs.FilterLogEventsOutput), args.Error(1)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

// TestFullWorkflowIntegration tests the complete workflow from CloudWatch logs to final JSON output
func TestFullWorkflowIntegration(t *testing.T) {
	tests := []struct {
		name           string
		mockLogs       []types.FilteredLogEvent
		expectedStats  int // number of path stats expected
		expectedPaths  []string
		expectError    bool
	}{
		{
			name: "complete workflow with matched request pairs",
			mockLogs: []types.FilteredLogEvent{
				{
					EventId:   stringPtr("event1"),
					Message:   stringPtr(`Started GET "/users/123" for 127.0.0.1 at 2025-07-10 17:28:13 +0900 [session-123]`),
					Timestamp: int64Ptr(1672531200000),
				},
				{
					EventId:   stringPtr("event2"),
					Message:   stringPtr(`Completed 200 OK in 150ms (Views: 100.0ms | ActiveRecord: 50.0ms) [session-123]`),
					Timestamp: int64Ptr(1672531200150),
				},
				{
					EventId:   stringPtr("event3"),
					Message:   stringPtr(`Started POST "/api/v1/orders" for 127.0.0.1 at 2025-07-10 17:28:14 +0900 [session-456]`),
					Timestamp: int64Ptr(1672531201000),
				},
				{
					EventId:   stringPtr("event4"),
					Message:   stringPtr(`Completed 201 Created in 139ms (Views: 80.0ms | ActiveRecord: 59.0ms) [session-456]`),
					Timestamp: int64Ptr(1672531201139),
				},
			},
			expectedStats: 2,
			expectedPaths: []string{"/users/:id", "/api/v1/orders"},
			expectError:   false,
		},
		{
			name: "workflow with unmatched started logs",
			mockLogs: []types.FilteredLogEvent{
				{
					EventId:   stringPtr("event1"),
					Message:   stringPtr(`Started GET "/users/123" for 127.0.0.1 at 2025-07-10 17:28:13 +0900 [session-123]`),
					Timestamp: int64Ptr(1672531200000),
				},
				{
					EventId:   stringPtr("event2"),
					Message:   stringPtr(`Started POST "/api/v1/orders" for 127.0.0.1 at 2025-07-10 17:28:14 +0900 [session-456]`),
					Timestamp: int64Ptr(1672531201000),
				},
				// No completed logs
			},
			expectedStats: 0,
			expectedPaths: []string{},
			expectError:   false,
		},
		{
			name: "workflow with excluded paths",
			mockLogs: []types.FilteredLogEvent{
				{
					EventId:   stringPtr("event1"),
					Message:   stringPtr(`Started GET "/rails/active_storage/blobs/123" for 127.0.0.1 at 2025-07-10 17:28:13 +0900 [session-123]`),
					Timestamp: int64Ptr(1672531200000),
				},
				{
					EventId:   stringPtr("event2"),
					Message:   stringPtr(`Completed 200 OK in 150ms (Views: 100.0ms | ActiveRecord: 50.0ms) [session-123]`),
					Timestamp: int64Ptr(1672531200150),
				},
				{
					EventId:   stringPtr("event3"),
					Message:   stringPtr(`Started GET "/users/456" for 127.0.0.1 at 2025-07-10 17:28:14 +0900 [session-456]`),
					Timestamp: int64Ptr(1672531201000),
				},
				{
					EventId:   stringPtr("event4"),
					Message:   stringPtr(`Completed 200 OK in 139ms (Views: 80.0ms | ActiveRecord: 59.0ms) [session-456]`),
					Timestamp: int64Ptr(1672531201139),
				},
			},
			expectedStats: 1, // Only /users/:id should be included
			expectedPaths: []string{"/users/:id"},
			expectError:   false,
		},
		{
			name: "workflow with pagination simulation",
			mockLogs: []types.FilteredLogEvent{
				{
					EventId:   stringPtr("event1"),
					Message:   stringPtr(`Started GET "/users/1" for 127.0.0.1 at 2025-07-10 17:28:13 +0900 [session-111]`),
					Timestamp: int64Ptr(1672531200000),
				},
				{
					EventId:   stringPtr("event2"),
					Message:   stringPtr(`Completed 200 OK in 100ms (Views: 50.0ms | ActiveRecord: 50.0ms) [session-111]`),
					Timestamp: int64Ptr(1672531200100),
				},
				{
					EventId:   stringPtr("event3"),
					Message:   stringPtr(`Started GET "/users/2" for 127.0.0.1 at 2025-07-10 17:28:14 +0900 [session-222]`),
					Timestamp: int64Ptr(1672531201000),
				},
				{
					EventId:   stringPtr("event4"),
					Message:   stringPtr(`Completed 200 OK in 200ms (Views: 100.0ms | ActiveRecord: 100.0ms) [session-222]`),
					Timestamp: int64Ptr(1672531201200),
				},
			},
			expectedStats: 1, // Both should be normalized to /users/:id
			expectedPaths: []string{"/users/:id"},
			expectError:   false,
		},
		{
			name: "empty log response",
			mockLogs: []types.FilteredLogEvent{},
			expectedStats: 0,
			expectedPaths: []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock CloudWatch client
			mockAPI := new(MockCloudWatchLogsAPI)
			client := cloudwatch.NewClientWithAPI(mockAPI)

			// Mock CloudWatch response
			logGroupName := "test-log-group"
			startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
			endTime := time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)

			filterPattern := `?Started ?Completed`
			expectedInput := &cloudwatchlogs.FilterLogEventsInput{
				LogGroupName:  &logGroupName,
				StartTime:     int64Ptr(startTime.UnixMilli()),
				EndTime:       int64Ptr(endTime.UnixMilli()),
				FilterPattern: &filterPattern,
			}

			mockResponse := &cloudwatchlogs.FilterLogEventsOutput{
				Events: tt.mockLogs,
			}
			mockAPI.On("FilterLogEvents", mock.Anything, expectedInput).Return(mockResponse, nil)

			// Execute CloudWatch log retrieval
			events, err := client.FilterLogEventsWithPagination(context.Background(), logGroupName, startTime, endTime)
			require.NoError(t, err)

			// Convert CloudWatch events to LogEvent models
			logEvents := make([]*models.LogEvent, len(events))
			for i, event := range events {
				logEvents[i] = &models.LogEvent{
					ID:        *event.EventId,
					Message:   *event.Message,
					Timestamp: time.UnixMilli(*event.Timestamp),
				}
			}

			// Execute analysis pipeline
			analyzer := analyzer.NewAnalyzer()
			result := analyzer.AnalyzeLogEvents(logEvents, startTime, endTime)

			if tt.expectError {
				return
			}

			assert.Len(t, result.PathMetrics, tt.expectedStats)

			// Verify expected paths are present
			for _, expectedPath := range tt.expectedPaths {
				found := false
				for path := range result.PathMetrics {
					if path == expectedPath {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected path %s not found in results", expectedPath)
			}

			mockAPI.AssertExpectations(t)
		})
	}
}

// TestWorkflowWithPaginationIntegration tests the full workflow with CloudWatch pagination
func TestWorkflowWithPaginationIntegration(t *testing.T) {
	mockAPI := new(MockCloudWatchLogsAPI)
	client := cloudwatch.NewClientWithAPI(mockAPI)

	logGroupName := "test-log-group"
	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)

	// Setup pagination scenario
	filterPattern := `?Started ?Completed`
	
	// First page
	firstPageInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  &logGroupName,
		StartTime:     int64Ptr(startTime.UnixMilli()),
		EndTime:       int64Ptr(endTime.UnixMilli()),
		FilterPattern: &filterPattern,
	}
	firstPageOutput := &cloudwatchlogs.FilterLogEventsOutput{
		Events: []types.FilteredLogEvent{
			{
				EventId:   stringPtr("event1"),
				Message:   stringPtr(`Started GET "/users/123" for 127.0.0.1 at 2025-07-10 17:28:13 +0900 [session-abc]`),
				Timestamp: int64Ptr(1672531200000),
			},
		},
		NextToken: stringPtr("next-token"),
	}

	// Second page
	secondPageInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  &logGroupName,
		StartTime:     int64Ptr(startTime.UnixMilli()),
		EndTime:       int64Ptr(endTime.UnixMilli()),
		NextToken:     stringPtr("next-token"),
		FilterPattern: &filterPattern,
	}
	secondPageOutput := &cloudwatchlogs.FilterLogEventsOutput{
		Events: []types.FilteredLogEvent{
			{
				EventId:   stringPtr("event2"),
				Message:   stringPtr(`Completed 200 OK in 150ms (Views: 100.0ms | ActiveRecord: 50.0ms) [session-abc]`),
				Timestamp: int64Ptr(1672531200150),
			},
		},
	}

	mockAPI.On("FilterLogEvents", mock.Anything, firstPageInput).Return(firstPageOutput, nil)
	mockAPI.On("FilterLogEvents", mock.Anything, secondPageInput).Return(secondPageOutput, nil)

	// Execute CloudWatch log retrieval with pagination
	events, err := client.FilterLogEventsWithPagination(context.Background(), logGroupName, startTime, endTime)
	require.NoError(t, err)
	assert.Len(t, events, 2)

	// Convert CloudWatch events to LogEvent models
	logEvents := make([]*models.LogEvent, len(events))
	for i, event := range events {
		logEvents[i] = &models.LogEvent{
			ID:   *event.EventId,
			Message:   *event.Message,
			Timestamp: time.UnixMilli(*event.Timestamp),
		}
	}

	// Execute analysis pipeline
	analyzer := analyzer.NewAnalyzer()
	result := analyzer.AnalyzeLogEvents(logEvents, startTime, endTime)

	// Should have 1 result for /users/:id with matched pair
	assert.Len(t, result.PathMetrics, 1)
	pathMetric, exists := result.PathMetrics["/users/:id"]
	assert.True(t, exists)
	assert.NotNil(t, pathMetric)
	assert.Equal(t, "/users/:id", pathMetric.Path)
	assert.Equal(t, 1, pathMetric.Count)
	assert.Equal(t, 150.0, pathMetric.AverageTime)

	mockAPI.AssertExpectations(t)
}

// TestErrorHandlingIntegration tests error scenarios in the full workflow
func TestErrorHandlingIntegration(t *testing.T) {
	tests := []struct {
		name        string
		mockError   error
		expectError bool
		errorMsg    string
	}{
		{
			name:        "CloudWatch API error",
			mockError:   assert.AnError,
			expectError: true,
			errorMsg:    "assert.AnError general error for testing",
		},
		{
			name:        "successful call with no errors",
			mockError:   nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockCloudWatchLogsAPI)
			client := cloudwatch.NewClientWithAPI(mockAPI)

			logGroupName := "test-log-group"
			startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
			endTime := time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)

			filterPattern := `?Started ?Completed`
			expectedInput := &cloudwatchlogs.FilterLogEventsInput{
				LogGroupName:  &logGroupName,
				StartTime:     int64Ptr(startTime.UnixMilli()),
				EndTime:       int64Ptr(endTime.UnixMilli()),
				FilterPattern: &filterPattern,
			}

			if tt.mockError != nil {
				mockAPI.On("FilterLogEvents", mock.Anything, expectedInput).Return((*cloudwatchlogs.FilterLogEventsOutput)(nil), tt.mockError)
			} else {
				mockResponse := &cloudwatchlogs.FilterLogEventsOutput{
					Events: []types.FilteredLogEvent{},
				}
				mockAPI.On("FilterLogEvents", mock.Anything, expectedInput).Return(mockResponse, nil)
			}

			// Execute CloudWatch log retrieval
			events, err := client.FilterLogEventsWithPagination(context.Background(), logGroupName, startTime, endTime)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, events)
			} else {
				assert.NoError(t, err)

				// Continue with analysis pipeline for successful cases
				// Convert CloudWatch events to LogEvent models
				logEvents := make([]*models.LogEvent, len(events))
				for i, event := range events {
					logEvents[i] = &models.LogEvent{
						ID:   *event.EventId,
						Message:   *event.Message,
						Timestamp: time.UnixMilli(*event.Timestamp),
					}
				}

				analyzer := analyzer.NewAnalyzer()
				result := analyzer.AnalyzeLogEvents(logEvents, startTime, endTime)
				assert.NotNil(t, result)
				assert.NotNil(t, result.PathMetrics)
			}

			mockAPI.AssertExpectations(t)
		})
	}
}

// TestTimeZoneHandlingIntegration tests JST to UTC conversion in the full workflow
func TestTimeZoneHandlingIntegration(t *testing.T) {
	mockAPI := new(MockCloudWatchLogsAPI)
	client := cloudwatch.NewClientWithAPI(mockAPI)

	// Test JST input times
	jst, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	jstStart := time.Date(2023, 1, 1, 12, 0, 0, 0, jst) // 12:00 JST
	jstEnd := time.Date(2023, 1, 1, 13, 0, 0, 0, jst)   // 13:00 JST

	// Expected UTC times
	utcStart := jstStart.UTC() // 03:00 UTC
	utcEnd := jstEnd.UTC()     // 04:00 UTC

	logGroupName := "test-log-group"
	filterPattern := `?Started ?Completed`

	expectedInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  &logGroupName,
		StartTime:     int64Ptr(utcStart.UnixMilli()),
		EndTime:       int64Ptr(utcEnd.UnixMilli()),
		FilterPattern: &filterPattern,
	}

	mockResponse := &cloudwatchlogs.FilterLogEventsOutput{
		Events: []types.FilteredLogEvent{
			{
				EventId:   stringPtr("event1"),
				Message:   stringPtr(`Started GET "/users/123" for 127.0.0.1 at 2025-07-10 17:28:13 +0900 [session-123]`),
				Timestamp: int64Ptr(utcStart.UnixMilli()),
			},
		},
	}
	mockAPI.On("FilterLogEvents", mock.Anything, expectedInput).Return(mockResponse, nil)

	// Execute with JST times (simulating CLI input after parsing)
	events, err := client.FilterLogEventsWithPagination(context.Background(), logGroupName, jstStart, jstEnd)
	require.NoError(t, err)
	assert.Len(t, events, 1)

	// Verify that the CloudWatch API was called with correct UTC timestamps
	mockAPI.AssertExpectations(t)
}

// TestSessionBasedMatchingIntegration tests session-based log matching in the full workflow
func TestSessionBasedMatchingIntegration(t *testing.T) {
	mockAPI := new(MockCloudWatchLogsAPI)
	client := cloudwatch.NewClientWithAPI(mockAPI)

	logGroupName := "test-log-group"
	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)

	// Create logs with interleaved sessions (not chronological order)
	mockLogs := []types.FilteredLogEvent{
		{
			EventId:   stringPtr("event1"),
			Message:   stringPtr(`Started GET "/users/1" for 127.0.0.1 at 2025-07-10 17:28:13 +0900 [session-abc]`),
			Timestamp: int64Ptr(1672531200000),
		},
		{
			EventId:   stringPtr("event2"),
			Message:   stringPtr(`Started GET "/users/2" for 127.0.0.1 at 2025-07-10 17:28:14 +0900 [session-xyz]`),
			Timestamp: int64Ptr(1672531201000),
		},
		{
			EventId:   stringPtr("event3"),
			Message:   stringPtr(`Completed 200 OK in 100ms (Views: 50.0ms | ActiveRecord: 50.0ms) [session-abc]`),
			Timestamp: int64Ptr(1672531202000),
		},
		{
			EventId:   stringPtr("event4"),
			Message:   stringPtr(`Completed 200 OK in 200ms (Views: 100.0ms | ActiveRecord: 100.0ms) [session-xyz]`),
			Timestamp: int64Ptr(1672531203000),
		},
	}

	filterPattern := `?Started ?Completed`
	expectedInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  &logGroupName,
		StartTime:     int64Ptr(startTime.UnixMilli()),
		EndTime:       int64Ptr(endTime.UnixMilli()),
		FilterPattern: &filterPattern,
	}

	mockResponse := &cloudwatchlogs.FilterLogEventsOutput{
		Events: mockLogs,
	}
	mockAPI.On("FilterLogEvents", mock.Anything, expectedInput).Return(mockResponse, nil)

	// Execute CloudWatch log retrieval
	events, err := client.FilterLogEventsWithPagination(context.Background(), logGroupName, startTime, endTime)
	require.NoError(t, err)

	// Convert CloudWatch events to LogEvent models
	logEvents := make([]*models.LogEvent, len(events))
	for i, event := range events {
		logEvents[i] = &models.LogEvent{
			ID:   *event.EventId,
			Message:   *event.Message,
			Timestamp: time.UnixMilli(*event.Timestamp),
		}
	}

	// Execute analysis pipeline
	analyzer := analyzer.NewAnalyzer()
	result := analyzer.AnalyzeLogEvents(logEvents, startTime, endTime)

	// Should have 1 result for /users/:id with 2 matched pairs
	assert.Len(t, result.PathMetrics, 1)
	pathMetric, exists := result.PathMetrics["/users/:id"]
	assert.True(t, exists)
	assert.NotNil(t, pathMetric)
	assert.Equal(t, "/users/:id", pathMetric.Path)
	assert.Equal(t, 2, pathMetric.Count)
	assert.Equal(t, 150.0, pathMetric.AverageTime) // (100+200)/2

	mockAPI.AssertExpectations(t)
}