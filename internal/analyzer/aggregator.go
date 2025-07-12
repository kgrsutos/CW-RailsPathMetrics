package analyzer

import (
	"time"

	"github.com/kgrsutos/cw-railspathmetrics/internal/config"
	"github.com/kgrsutos/cw-railspathmetrics/internal/models"
)

// Aggregator handles aggregation of log entries into metrics
type Aggregator struct {
	pathExcluder *config.PathExcluder
}

// NewAggregator creates a new Aggregator instance with default path exclusions
func NewAggregator() *Aggregator {
	return &Aggregator{
		pathExcluder: config.NewDefaultPathExcluder(),
	}
}

// NewAggregatorWithConfig creates a new Aggregator instance with a config file
func NewAggregatorWithConfig(configPath string) (*Aggregator, error) {
	pathExcluder, err := config.NewPathExcluder(configPath)
	if err != nil {
		return nil, err
	}

	return &Aggregator{
		pathExcluder: pathExcluder,
	}, nil
}

// NewAggregatorWithPathExcluder creates a new Aggregator instance with a given PathExcluder
func NewAggregatorWithPathExcluder(pathExcluder *config.PathExcluder) *Aggregator {
	return &Aggregator{
		pathExcluder: pathExcluder,
	}
}

// MatchRequestPairs matches Started and Completed log entries by their SessionID
func (a *Aggregator) MatchRequestPairs(entries []*models.LogEntry) []*models.RequestPair {
	pairs := make([]*models.RequestPair, 0)
	startedLogs := make(map[string]*models.LogEntry)

	for _, entry := range entries {
		if entry.Type == "Started" {
			// Store Started logs by SessionID
			if entry.SessionID != "" {
				startedLogs[entry.SessionID] = entry
			}
		} else if entry.Type == "Completed" && entry.SessionID != "" {
			// Match with Started log with the same SessionID
			if started, exists := startedLogs[entry.SessionID]; exists {
				pairs = append(pairs, &models.RequestPair{
					Started:   started,
					Completed: entry,
				})
				// Remove matched Started log to avoid duplicate matches
				delete(startedLogs, entry.SessionID)
			}
		}
	}

	return pairs
}

// AggregateMetrics aggregates request pairs into path metrics
func (a *Aggregator) AggregateMetrics(pairs []*models.RequestPair, normalizer *Normalizer) map[string]*models.PathMetrics {
	pathMetrics := make(map[string]*models.PathMetrics)

	for _, pair := range pairs {
		// Check if the path should be excluded
		if a.pathExcluder.ShouldExclude(pair.Started.Path) {
			continue
		}

		// Normalize the path
		normalizedPath := normalizer.NormalizePath(pair.Started.Path)

		// Get or create path metrics
		metrics, exists := pathMetrics[normalizedPath]
		if !exists {
			metrics = &models.PathMetrics{
				Path:        normalizedPath,
				Count:       0,
				AverageTime: 0,
				MinTime:     0,
				MaxTime:     0,
				StatusCodes: make(map[int]int),
				Methods:     make(map[string]int),
			}
			pathMetrics[normalizedPath] = metrics
		}

		// Update metrics
		metrics.Count++

		// Update timing metrics
		duration := pair.Completed.Duration
		if metrics.Count == 1 {
			metrics.MinTime = duration
			metrics.MaxTime = duration
			metrics.AverageTime = float64(duration)
		} else {
			if duration < metrics.MinTime {
				metrics.MinTime = duration
			}
			if duration > metrics.MaxTime {
				metrics.MaxTime = duration
			}
			// Calculate new average: (old_avg * (count-1) + new_value) / count
			metrics.AverageTime = (metrics.AverageTime*float64(metrics.Count-1) + float64(duration)) / float64(metrics.Count)
		}

		// Update status codes
		metrics.StatusCodes[pair.Completed.StatusCode]++

		// Update methods
		metrics.Methods[pair.Started.Method]++

		// Update view and DB durations if present
		if pair.Completed.ViewDuration > 0 {
			metrics.TotalViewDuration += pair.Completed.ViewDuration
		}
		if pair.Completed.DBDuration > 0 {
			metrics.TotalDBDuration += pair.Completed.DBDuration
		}
	}

	return pathMetrics
}

// AnalyzeLogs performs complete analysis of log entries
func (a *Aggregator) AnalyzeLogs(entries []*models.LogEntry, normalizer *Normalizer, startTime, endTime time.Time) *models.AnalysisResult {
	// Match Started and Completed logs
	pairs := a.MatchRequestPairs(entries)

	// Aggregate metrics
	pathMetrics := a.AggregateMetrics(pairs, normalizer)

	return &models.AnalysisResult{
		StartTime:   startTime,
		EndTime:     endTime,
		TotalLogs:   len(entries),
		PathMetrics: pathMetrics,
	}
}
