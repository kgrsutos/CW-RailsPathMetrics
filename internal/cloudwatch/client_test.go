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

			expectedInput := &cloudwatchlogs.FilterLogEventsInput{
				LogGroupName: &tt.logGroupName,
				StartTime:    int64Ptr(tt.startTime.UnixMilli()),
				EndTime:      int64Ptr(tt.endTime.UnixMilli()),
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
	firstPageInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: &logGroupName,
		StartTime:    int64Ptr(startTime.UnixMilli()),
		EndTime:      int64Ptr(endTime.UnixMilli()),
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
		LogGroupName: &logGroupName,
		StartTime:    int64Ptr(startTime.UnixMilli()),
		EndTime:      int64Ptr(endTime.UnixMilli()),
		NextToken:    stringPtr("next-token"),
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
