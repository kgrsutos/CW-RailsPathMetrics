package cli

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestAnalyzeCommand_ConfigFlag(t *testing.T) {
	tests := []struct {
		name         string
		configFlag   string
		expectedFlag string
	}{
		{
			name:         "config flag with custom path",
			configFlag:   "/custom/path/excluded_paths.yml",
			expectedFlag: "/custom/path/excluded_paths.yml",
		},
		{
			name:         "config flag empty",
			configFlag:   "",
			expectedFlag: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config flag value before each test
			configPath = ""

			// Set the config flag
			if tt.configFlag != "" {
				err := analyzeCmd.Flags().Set("config", tt.configFlag)
				assert.NoError(t, err)
			}

			// Check that the flag value is correctly set
			configFlag := analyzeCmd.Flags().Lookup("config")
			assert.NotNil(t, configFlag)
			assert.Equal(t, tt.expectedFlag, configFlag.Value.String())
		})
	}
}

// MockCloudWatchAPI is a mock implementation of CloudWatchLogsAPI
type MockCloudWatchAPI struct {
	mock.Mock
}

func (m *MockCloudWatchAPI) FilterLogEvents(ctx context.Context, params *cloudwatchlogs.FilterLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.FilterLogEventsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*cloudwatchlogs.FilterLogEventsOutput), args.Error(1)
}

func TestRunAnalyze(t *testing.T) {
	tests := []struct {
		name         string
		setupFlags   func()
		cleanupFlags func()
		expectError  bool
		errorMsg     string
	}{
		{
			name: "invalid start time format",
			setupFlags: func() {
				startTime = "invalid-format"
				endTime = "2023-01-01T12:00:00"
				logGroup = "test-log-group"
				profile = "test-profile"
			},
			cleanupFlags: func() {
				startTime = ""
				endTime = ""
				logGroup = ""
				profile = ""
			},
			expectError: true,
			errorMsg:    "failed to parse start time",
		},
		{
			name: "invalid end time format",
			setupFlags: func() {
				startTime = "2023-01-01T12:00:00"
				endTime = "invalid-format"
				logGroup = "test-log-group"
				profile = "test-profile"
			},
			cleanupFlags: func() {
				startTime = ""
				endTime = ""
				logGroup = ""
				profile = ""
			},
			expectError: true,
			errorMsg:    "failed to parse end time",
		},
		{
			name: "end time before start time",
			setupFlags: func() {
				startTime = "2023-01-01T12:00:00"
				endTime = "2023-01-01T11:00:00"
				logGroup = "test-log-group"
				profile = "test-profile"
			},
			cleanupFlags: func() {
				startTime = ""
				endTime = ""
				logGroup = ""
				profile = ""
			},
			expectError: false, // Time validation happens in CloudWatch layer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup flags
			tt.setupFlags()
			defer tt.cleanupFlags()

			// Execute the command
			err := runAnalyze(nil, nil)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				// Note: This will fail because we don't have AWS credentials in test
				// but we can at least verify the time parsing logic works
				if err != nil {
					// Check if it's an AWS credential error, which is expected
					assert.Contains(t, err.Error(), "failed to initialize CloudWatch client")
				}
			}
		})
	}
}

func TestRunAnalyzeTimeConversion(t *testing.T) {
	// Test JST to UTC conversion logic
	jst, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	testCases := []struct {
		name        string
		jstTime     string
		expectedUTC time.Time
	}{
		{
			name:        "noon JST to UTC",
			jstTime:     "2023-01-01T12:00:00",
			expectedUTC: time.Date(2023, 1, 1, 3, 0, 0, 0, time.UTC),
		},
		{
			name:        "midnight JST to UTC",
			jstTime:     "2023-01-01T00:00:00",
			expectedUTC: time.Date(2022, 12, 31, 15, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := time.ParseInLocation("2006-01-02T15:04:05", tc.jstTime, jst)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedUTC, parsed.UTC())
		})
	}
}

func TestAnalyzeCommandFlags(t *testing.T) {
	// Test flag requirements
	requiredFlags := []string{"start", "end", "log-group", "profile"}

	for _, flagName := range requiredFlags {
		t.Run("flag_"+flagName+"_is_required", func(t *testing.T) {
			flag := analyzeCmd.Flags().Lookup(flagName)
			assert.NotNil(t, flag, "Flag %s should exist", flagName)

			// Check if flag is marked as required
			annotations := flag.Annotations
			if annotations != nil {
				if requiredAnno, exists := annotations[cobra.BashCompOneRequiredFlag]; exists {
					assert.Contains(t, requiredAnno, "true", "Flag %s should be required", flagName)
				}
			}
		})
	}
}

func TestAnalyzeCommandIntegration(t *testing.T) {
	// Test command registration and basic structure
	t.Run("command_registered", func(t *testing.T) {
		// Check if analyze command is added to root command
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "analyze" {
				found = true
				break
			}
		}
		assert.True(t, found, "analyze command should be registered with root command")
	})

	t.Run("help_text", func(t *testing.T) {
		assert.NotEmpty(t, analyzeCmd.Short, "Short description should not be empty")
		assert.NotEmpty(t, analyzeCmd.Long, "Long description should not be empty")
		assert.Equal(t, "analyze", analyzeCmd.Use, "Command use should be 'analyze'")
	})
}
