package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/kgrsutos/cw-railspathmetrics/internal/analyzer"
	"github.com/kgrsutos/cw-railspathmetrics/internal/cloudwatch"
	"github.com/kgrsutos/cw-railspathmetrics/internal/models"
)

var (
	startTime  string
	endTime    string
	logGroup   string
	profile    string
	configPath string
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze CloudWatch logs for Rails request metrics",
	Long:  `Analyze CloudWatch logs to aggregate request metrics by path.`,
	RunE:  runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	analyzeCmd.Flags().StringVar(&startTime, "start", "", "Start time in JST (required, format: 2006-01-02T15:04:05)")
	analyzeCmd.Flags().StringVar(&endTime, "end", "", "End time in JST (required, format: 2006-01-02T15:04:05)")
	analyzeCmd.Flags().StringVar(&logGroup, "log-group", "", "CloudWatch Logs log group name (required)")
	analyzeCmd.Flags().StringVar(&profile, "profile", "", "AWS profile name (required)")
	analyzeCmd.Flags().StringVar(&configPath, "config", "", "Path exclusion configuration file (optional, defaults to config/excluded_paths.yml)")

	if err := analyzeCmd.MarkFlagRequired("start"); err != nil {
		slog.Error("Failed to mark start flag as required", "error", err)
	}
	if err := analyzeCmd.MarkFlagRequired("end"); err != nil {
		slog.Error("Failed to mark end flag as required", "error", err)
	}
	if err := analyzeCmd.MarkFlagRequired("log-group"); err != nil {
		slog.Error("Failed to mark log-group flag as required", "error", err)
	}
	if err := analyzeCmd.MarkFlagRequired("profile"); err != nil {
		slog.Error("Failed to mark profile flag as required", "error", err)
	}
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	slog.Info("Starting analysis",
		"start", startTime,
		"end", endTime,
		"logGroup", logGroup,
		"profile", profile,
		"config", configPath,
	)

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return fmt.Errorf("failed to load JST location: %w", err)
	}

	start, err := time.ParseInLocation("2006-01-02T15:04:05", startTime, jst)
	if err != nil {
		return fmt.Errorf("failed to parse start time: %w", err)
	}

	end, err := time.ParseInLocation("2006-01-02T15:04:05", endTime, jst)
	if err != nil {
		return fmt.Errorf("failed to parse end time: %w", err)
	}

	slog.Info("Parsed time range",
		"startUTC", start.UTC(),
		"endUTC", end.UTC(),
	)

	// Initialize CloudWatch client
	ctx := context.Background()
	client, err := cloudwatch.NewClient(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to initialize CloudWatch client: %w", err)
	}

	// Fetch log events
	slog.Info("Fetching log events from CloudWatch")
	events, err := client.FilterLogEventsWithPagination(ctx, logGroup, start, end)
	if err != nil {
		return fmt.Errorf("failed to fetch log events: %w", err)
	}

	// Convert CloudWatch events to our LogEvent model
	var logEvents []*models.LogEvent
	for _, event := range events {
		if event.EventId != nil && event.Message != nil && event.Timestamp != nil {
			logEvents = append(logEvents, &models.LogEvent{
				ID:        *event.EventId,
				Message:   *event.Message,
				Timestamp: time.UnixMilli(*event.Timestamp),
			})
		}
	}

	slog.Info("Fetched log events", "count", len(logEvents))

	// Initialize analyzer
	var analyzerInstance *analyzer.Analyzer
	if configPath != "" {
		analyzerInstance, err = analyzer.NewAnalyzerWithConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to initialize analyzer with config: %w", err)
		}
	} else {
		analyzerInstance = analyzer.NewAnalyzer()
	}

	// Analyze log events
	result := analyzerInstance.AnalyzeLogEvents(logEvents, start.UTC(), end.UTC())

	// Output JSON results
	err = analyzerInstance.OutputJSON(result, os.Stdout)
	if err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	return nil
}
