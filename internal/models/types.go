package models

import "time"

// LogEvent represents a CloudWatch log event
type LogEvent struct {
	ID        string
	Message   string
	Timestamp time.Time
}

// LogEntry represents a parsed Rails log entry
type LogEntry struct {
	Type         string    // "Started" or "Completed"
	Method       string    // HTTP method (GET, POST, etc.) - only for Started logs
	Path         string    // Request path - only for Started logs
	Timestamp    time.Time // Log timestamp - only for Started logs
	StatusCode   int       // HTTP status code - only for Completed logs
	StatusText   string    // Status text (OK, Not Found, etc.) - only for Completed logs
	Duration     int       // Total duration in milliseconds - only for Completed logs
	ViewDuration float64   // View rendering duration - only for Completed logs
	DBDuration   float64   // ActiveRecord duration - only for Completed logs
	SessionID    string    // Session identifier - only for Completed logs
}

// PathMetrics represents aggregated metrics for a specific path
type PathMetrics struct {
	Path              string         `json:"path"`
	Count             int            `json:"count"`
	AverageTime       float64        `json:"average_time_ms"`
	MinTime           int            `json:"min_time_ms"`
	MaxTime           int            `json:"max_time_ms"`
	StatusCodes       map[int]int    `json:"status_codes"`
	Methods           map[string]int `json:"methods"`
	TotalViewDuration float64        `json:"total_view_duration_ms,omitempty"`
	TotalDBDuration   float64        `json:"total_db_duration_ms,omitempty"`
}

// AnalysisResult represents the final analysis output
type AnalysisResult struct {
	StartTime   time.Time               `json:"start_time"`
	EndTime     time.Time               `json:"end_time"`
	TotalLogs   int                     `json:"total_logs_analyzed"`
	PathMetrics map[string]*PathMetrics `json:"path_metrics"`
}

// RequestPair represents a matched Started and Completed log pair
type RequestPair struct {
	Started   *LogEntry
	Completed *LogEntry
}

