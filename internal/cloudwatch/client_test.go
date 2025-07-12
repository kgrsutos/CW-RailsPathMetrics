package cloudwatch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCloudWatchLogsAPI struct {
	mock.Mock
}

func (m *MockCloudWatchLogsAPI) FilterLogEvents(ctx context.Context, params *cloudwatchlogs.FilterLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.FilterLogEventsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudwatchlogs.FilterLogEventsOutput), args.Error(1)
}

func TestClient_FilterLogEvents(t *testing.T) {
	tests := []struct {
		name           string
		logGroupName   string
		startTime      time.Time
		endTime        time.Time
		mockResponse   *cloudwatchlogs.FilterLogEventsOutput
		mockError      error
		expectedEvents []types.FilteredLogEvent
		expectedError  error
	}{
		{
			name:         "successful log retrieval",
			logGroupName: "test-log-group",
			startTime:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			endTime:      time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC),
			mockResponse: &cloudwatchlogs.FilterLogEventsOutput{
				Events: []types.FilteredLogEvent{
					{
						EventId:   stringPtr("event1"),
						Message:   stringPtr("Started GET \"/users/123\" for 127.0.0.1"),
						Timestamp: int64Ptr(1672531200000),
					},
					{
						EventId:   stringPtr("event2"),
						Message:   stringPtr("Completed 200 OK in 150ms"),
						Timestamp: int64Ptr(1672531200150),
					},
				},
			},
			expectedEvents: []types.FilteredLogEvent{
				{
					EventId:   stringPtr("event1"),
					Message:   stringPtr("Started GET \"/users/123\" for 127.0.0.1"),
					Timestamp: int64Ptr(1672531200000),
				},
				{
					EventId:   stringPtr("event2"),
					Message:   stringPtr("Completed 200 OK in 150ms"),
					Timestamp: int64Ptr(1672531200150),
				},
			},
		},
		{
			name:         "empty log group",
			logGroupName: "empty-log-group",
			startTime:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			endTime:      time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC),
			mockResponse: &cloudwatchlogs.FilterLogEventsOutput{
				Events: []types.FilteredLogEvent{},
			},
			expectedEvents: []types.FilteredLogEvent{},
		},
		{
			name:          "API error",
			logGroupName:  "test-log-group",
			startTime:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			endTime:       time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC),
			mockError:     errors.New("access denied"),
			expectedError: errors.New("access denied"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := new(MockCloudWatchLogsAPI)
			client := &Client{
				api: mockAPI,
			}

			filterPattern := `?Started ?Completed`
			expectedInput := &cloudwatchlogs.FilterLogEventsInput{
				LogGroupName:  &tt.logGroupName,
				StartTime:     int64Ptr(tt.startTime.UnixMilli()),
				EndTime:       int64Ptr(tt.endTime.UnixMilli()),
				FilterPattern: &filterPattern,
			}

			mockAPI.On("FilterLogEvents", mock.Anything, expectedInput).Return(tt.mockResponse, tt.mockError)

			events, err := client.FilterLogEvents(context.Background(), tt.logGroupName, tt.startTime, tt.endTime)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedEvents, events)
			}

			mockAPI.AssertExpectations(t)
		})
	}
}

func TestClient_FilterLogEventsWithPagination(t *testing.T) {
	mockAPI := new(MockCloudWatchLogsAPI)
	client := &Client{
		api: mockAPI,
	}

	logGroupName := "test-log-group"
	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)

	// First page
	filterPattern := `?Started ?Completed`
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
				Message:   stringPtr("Started GET \"/users/123\""),
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
				Message:   stringPtr("Completed 200 OK"),
				Timestamp: int64Ptr(1672531200150),
			},
		},
	}

	mockAPI.On("FilterLogEvents", mock.Anything, firstPageInput).Return(firstPageOutput, nil)
	mockAPI.On("FilterLogEvents", mock.Anything, secondPageInput).Return(secondPageOutput, nil)

	events, err := client.FilterLogEventsWithPagination(context.Background(), logGroupName, startTime, endTime)

	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "event1", *events[0].EventId)
	assert.Equal(t, "event2", *events[1].EventId)

	mockAPI.AssertExpectations(t)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func TestClient_FilterLogEventsWithPagination_ErrorHandling(t *testing.T) {
	mockAPI := new(MockCloudWatchLogsAPI)
	client := &Client{
		api: mockAPI,
	}

	logGroupName := "test-log-group"
	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)

	// Mock API error on first call
	filterPattern := `?Started ?Completed`
	expectedInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  &logGroupName,
		StartTime:     int64Ptr(startTime.UnixMilli()),
		EndTime:       int64Ptr(endTime.UnixMilli()),
		FilterPattern: &filterPattern,
	}
	mockAPI.On("FilterLogEvents", mock.Anything, expectedInput).Return((*cloudwatchlogs.FilterLogEventsOutput)(nil), errors.New("API error"))

	events, err := client.FilterLogEventsWithPagination(context.Background(), logGroupName, startTime, endTime)

	assert.Error(t, err)
	assert.Equal(t, "API error", err.Error())
	assert.Nil(t, events)

	mockAPI.AssertExpectations(t)
}

func TestClient_FilterLogEventsWithPagination_EmptyResponse(t *testing.T) {
	mockAPI := new(MockCloudWatchLogsAPI)
	client := &Client{
		api: mockAPI,
	}

	logGroupName := "empty-log-group"
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
		Events: []types.FilteredLogEvent{},
	}
	mockAPI.On("FilterLogEvents", mock.Anything, expectedInput).Return(mockResponse, nil)

	events, err := client.FilterLogEventsWithPagination(context.Background(), logGroupName, startTime, endTime)

	assert.NoError(t, err)
	assert.Empty(t, events)

	mockAPI.AssertExpectations(t)
}

func TestClient_FilterLogEventsWithPagination_MultiplePages(t *testing.T) {
	mockAPI := new(MockCloudWatchLogsAPI)
	client := &Client{
		api: mockAPI,
	}

	logGroupName := "test-log-group"
	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)

	// Setup 3 pages of results
	filterPattern := `?Started ?Completed`
	page1Input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  &logGroupName,
		StartTime:     int64Ptr(startTime.UnixMilli()),
		EndTime:       int64Ptr(endTime.UnixMilli()),
		FilterPattern: &filterPattern,
	}
	page1Output := &cloudwatchlogs.FilterLogEventsOutput{
		Events: []types.FilteredLogEvent{
			{EventId: stringPtr("event1"), Message: stringPtr("log1"), Timestamp: int64Ptr(1672531200000)},
		},
		NextToken: stringPtr("token1"),
	}

	page2Input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  &logGroupName,
		StartTime:     int64Ptr(startTime.UnixMilli()),
		EndTime:       int64Ptr(endTime.UnixMilli()),
		NextToken:     stringPtr("token1"),
		FilterPattern: &filterPattern,
	}
	page2Output := &cloudwatchlogs.FilterLogEventsOutput{
		Events: []types.FilteredLogEvent{
			{EventId: stringPtr("event2"), Message: stringPtr("log2"), Timestamp: int64Ptr(1672531200100)},
		},
		NextToken: stringPtr("token2"),
	}

	page3Input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  &logGroupName,
		StartTime:     int64Ptr(startTime.UnixMilli()),
		EndTime:       int64Ptr(endTime.UnixMilli()),
		NextToken:     stringPtr("token2"),
		FilterPattern: &filterPattern,
	}
	page3Output := &cloudwatchlogs.FilterLogEventsOutput{
		Events: []types.FilteredLogEvent{
			{EventId: stringPtr("event3"), Message: stringPtr("log3"), Timestamp: int64Ptr(1672531200200)},
		},
		// No NextToken - end of results
	}

	mockAPI.On("FilterLogEvents", mock.Anything, page1Input).Return(page1Output, nil)
	mockAPI.On("FilterLogEvents", mock.Anything, page2Input).Return(page2Output, nil)
	mockAPI.On("FilterLogEvents", mock.Anything, page3Input).Return(page3Output, nil)

	events, err := client.FilterLogEventsWithPagination(context.Background(), logGroupName, startTime, endTime)

	assert.NoError(t, err)
	assert.Len(t, events, 3)
	assert.Equal(t, "event1", *events[0].EventId)
	assert.Equal(t, "event2", *events[1].EventId)
	assert.Equal(t, "event3", *events[2].EventId)

	mockAPI.AssertExpectations(t)
}

func TestNewClientWithAPI(t *testing.T) {
	mockAPI := new(MockCloudWatchLogsAPI)
	client := NewClientWithAPI(mockAPI)

	assert.NotNil(t, client)
	assert.Equal(t, mockAPI, client.api)
}

func TestClient_FilterLogEvents_NilPointers(t *testing.T) {
	mockAPI := new(MockCloudWatchLogsAPI)
	client := &Client{
		api: mockAPI,
	}

	logGroupName := "test-log-group"
	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC)

	// Mock response with nil pointers (should be filtered out)
	mockResponse := &cloudwatchlogs.FilterLogEventsOutput{
		Events: []types.FilteredLogEvent{
			{
				EventId:   stringPtr("event1"),
				Message:   stringPtr("Started GET \"/users/123\""),
				Timestamp: int64Ptr(1672531200000),
			},
			{
				EventId:   nil, // Should be filtered out
				Message:   stringPtr("log with nil EventId"),
				Timestamp: int64Ptr(1672531200100),
			},
			{
				EventId:   stringPtr("event3"),
				Message:   nil, // Should be filtered out
				Timestamp: int64Ptr(1672531200200),
			},
		},
	}

	filterPattern := `?Started ?Completed`
	expectedInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  &logGroupName,
		StartTime:     int64Ptr(startTime.UnixMilli()),
		EndTime:       int64Ptr(endTime.UnixMilli()),
		FilterPattern: &filterPattern,
	}
	mockAPI.On("FilterLogEvents", mock.Anything, expectedInput).Return(mockResponse, nil)

	events, err := client.FilterLogEvents(context.Background(), logGroupName, startTime, endTime)

	assert.NoError(t, err)
	assert.Len(t, events, 3) // All events returned, filtering happens in CLI layer
	assert.Equal(t, "event1", *events[0].EventId)
	assert.Nil(t, events[1].EventId)
	assert.Nil(t, events[2].Message)

	mockAPI.AssertExpectations(t)
}

func TestInt64Ptr(t *testing.T) {
	value := int64(12345)
	ptr := int64Ptr(value)

	assert.NotNil(t, ptr)
	assert.Equal(t, value, *ptr)
}
