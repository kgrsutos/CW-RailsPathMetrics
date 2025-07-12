package cloudwatch

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// CloudWatchLogsAPI defines the interface for CloudWatch Logs operations
type CloudWatchLogsAPI interface {
	FilterLogEvents(ctx context.Context, params *cloudwatchlogs.FilterLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.FilterLogEventsOutput, error)
}

// Client wraps AWS CloudWatch Logs client
type Client struct {
	api CloudWatchLogsAPI
}

// NewClient creates a new CloudWatch client with AWS SDK configuration
func NewClient(ctx context.Context, profile string) (*Client, error) {
	var cfg aws.Config
	var err error

	if profile != "" {
		cfg, err = config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	} else {
		cfg, err = config.LoadDefaultConfig(ctx)
	}

	if err != nil {
		return nil, err
	}

	return &Client{
		api: cloudwatchlogs.NewFromConfig(cfg),
	}, nil
}

// NewClientWithAPI creates a new CloudWatch client with a custom API implementation
// This is primarily used for testing
func NewClientWithAPI(api CloudWatchLogsAPI) *Client {
	return &Client{
		api: api,
	}
}

// FilterLogEvents retrieves log events from CloudWatch Logs
func (c *Client) FilterLogEvents(ctx context.Context, logGroupName string, startTime, endTime time.Time) ([]types.FilteredLogEvent, error) {
	// Filter pattern to only fetch logs containing "Started" or "Completed"
	// This reduces data transfer and costs by filtering at CloudWatch level
	// Using regex pattern for unstructured Rails logs
	filterPattern := `?Started ?Completed`

	input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  &logGroupName,
		StartTime:     int64Ptr(startTime.UnixMilli()),
		EndTime:       int64Ptr(endTime.UnixMilli()),
		FilterPattern: &filterPattern,
	}

	output, err := c.api.FilterLogEvents(ctx, input)
	if err != nil {
		return nil, err
	}

	return output.Events, nil
}

// FilterLogEventsWithPagination retrieves all log events with pagination support
func (c *Client) FilterLogEventsWithPagination(ctx context.Context, logGroupName string, startTime, endTime time.Time) ([]types.FilteredLogEvent, error) {
	var allEvents []types.FilteredLogEvent
	var nextToken *string

	// Filter pattern to only fetch logs containing "Started" or "Completed"
	// This reduces data transfer and costs by filtering at CloudWatch level
	// Using regex pattern for unstructured Rails logs
	filterPattern := `?Started ?Completed`

	for {
		input := &cloudwatchlogs.FilterLogEventsInput{
			LogGroupName:  &logGroupName,
			StartTime:     int64Ptr(startTime.UnixMilli()),
			EndTime:       int64Ptr(endTime.UnixMilli()),
			NextToken:     nextToken,
			FilterPattern: &filterPattern,
		}

		output, err := c.api.FilterLogEvents(ctx, input)
		if err != nil {
			return nil, err
		}

		allEvents = append(allEvents, output.Events...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return allEvents, nil
}

// Helper function to create int64 pointer
func int64Ptr(i int64) *int64 {
	return &i
}
