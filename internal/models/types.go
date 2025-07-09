package models

import (
	"time"
)

// LogEvent represents a single log event from CloudWatch Logs
type LogEvent struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// RailsLogEntry represents a parsed Rails log entry
type RailsLogEntry struct {
	Type      LogType   `json:"type"`      // "started" or "completed"
	Method    string    `json:"method"`    // HTTP method (GET, POST, etc.)
	Path      string    `json:"path"`      // Request path
	SessionID string    `json:"session_id"` // Session identifier
	IP        string    `json:"ip"`        // Client IP address
	Timestamp time.Time `json:"timestamp"`
	
	// Completed log specific fields
	StatusCode    int     `json:"status_code,omitempty"`    // HTTP status code
	ResponseTime  float64 `json:"response_time,omitempty"`  // Response time in milliseconds
	ViewTime      float64 `json:"view_time,omitempty"`      // View rendering time in milliseconds
	DatabaseTime  float64 `json:"database_time,omitempty"`  // Database query time in milliseconds
	AllocatedSize int64   `json:"allocated_size,omitempty"` // Memory allocated in bytes
}

// LogType represents the type of Rails log entry
type LogType string

const (
	LogTypeStarted   LogType = "started"
	LogTypeCompleted LogType = "completed"
)

// PathMetric represents aggregated metrics for a specific path
type PathMetric struct {
	Path         string    `json:"path"`
	NormalizedPath string  `json:"normalized_path"`
	Method       string    `json:"method"`
	Count        int       `json:"count"`
	TotalTime    float64   `json:"total_time"`
	MinTime      float64   `json:"min_time"`
	MaxTime      float64   `json:"max_time"`
	AvgTime      float64   `json:"avg_time"`
	MedianTime   float64   `json:"median_time"`
	P95Time      float64   `json:"p95_time"`
	P99Time      float64   `json:"p99_time"`
	
	// Status code distribution
	StatusCodes map[int]int `json:"status_codes"`
	
	// Time range
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// MetricsResult represents the final output of the analysis
type MetricsResult struct {
	Summary Summary      `json:"summary"`
	Paths   []PathMetric `json:"paths"`
}

// Summary represents overall statistics
type Summary struct {
	TotalRequests     int       `json:"total_requests"`
	TotalPaths        int       `json:"total_paths"`
	TimeRange         TimeRange `json:"time_range"`
	AverageResponseTime float64 `json:"average_response_time"`
	TotalErrors       int       `json:"total_errors"`
	ErrorRate         float64   `json:"error_rate"`
}

// TimeRange represents a time range for analysis
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// RequestSession represents a matched pair of started and completed log entries
type RequestSession struct {
	Started   *RailsLogEntry `json:"started"`
	Completed *RailsLogEntry `json:"completed"`
	Duration  float64        `json:"duration"` // Duration in milliseconds
}
