package analyzer

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/kgrsutos/cw-railspathmetrics/internal/config"
	"github.com/kgrsutos/cw-railspathmetrics/internal/models"
)

// Analyzer coordinates the analysis of Rails log entries
type Analyzer struct {
	parser     *Parser
	normalizer *Normalizer
	aggregator *Aggregator
}

// NewAnalyzer creates a new Analyzer instance with default configuration
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		parser:     NewParser(),
		normalizer: NewNormalizer(),
		aggregator: NewAggregator(),
	}
}

// NewAnalyzerWithConfig creates a new Analyzer instance with custom configuration
func NewAnalyzerWithConfig(configPath string) (*Analyzer, error) {
	var aggregator *Aggregator
	var err error
	
	if configPath != "" {
		// Use specific config file
		aggregator, err = NewAggregatorWithConfig(configPath)
		if err != nil {
			return nil, err
		}
	} else {
		// Search for config file or use default
		pathExcluder, err := config.NewPathExcluderWithSearch()
		if err != nil {
			return nil, err
		}
		aggregator = NewAggregatorWithPathExcluder(pathExcluder)
	}
	
	return &Analyzer{
		parser:     NewParser(),
		normalizer: NewNormalizer(),
		aggregator: aggregator,
	}, nil
}

// AnalyzeLogEvents analyzes CloudWatch log events and returns aggregated metrics
func (a *Analyzer) AnalyzeLogEvents(logEvents []*models.LogEvent, startTime, endTime time.Time) *models.AnalysisResult {
	var logEntries []*models.LogEntry

	// Parse log events into log entries
	for _, logEvent := range logEvents {
		logEntry, err := a.parser.ParseLogEntry(logEvent.Message)
		if err != nil {
			// Skip invalid log entries
			continue
		}
		logEntries = append(logEntries, logEntry)
	}

	// Analyze log entries
	return a.aggregator.AnalyzeLogs(logEntries, a.normalizer, startTime, endTime)
}

// OutputJSON writes the analysis result as JSON to the provided writer
func (a *Analyzer) OutputJSON(result *models.AnalysisResult, writer io.Writer) error {
	// Convert to simplified format
	simplified := make([]*models.SimplifiedPathMetrics, 0, len(result.PathMetrics))

	for _, metrics := range result.PathMetrics {
		simplified = append(simplified, &models.SimplifiedPathMetrics{
			Path:      metrics.Path,
			Count:     metrics.Count,
			MaxTimeMs: metrics.MaxTime,
			MinTimeMs: metrics.MinTime,
			AvgTimeMs: fmt.Sprintf("%.0f", metrics.AverageTime),
		})
	}

	// Sort by count in descending order (highest count first)
	sort.Slice(simplified, func(i, j int) bool {
		return simplified[i].Count > simplified[j].Count
	})

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "    ")
	return encoder.Encode(simplified)
}
