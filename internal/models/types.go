package models

import "time"

type LogEntry struct {
	Timestamp time.Time
	Message   string
	SessionID string
	LogType   LogType
}

type LogType int

const (
	LogTypeUnknown LogType = iota
	LogTypeStarted
	LogTypeCompleted
)

type RequestInfo struct {
	SessionID      string
	Path           string
	StartTime      time.Time
	EndTime        *time.Time
	ProcessingTime *time.Duration
}

type PathMetrics struct {
	Path    string `json:"path"`
	Count   int    `json:"count"`
	MaxTime string `json:"max_time"`
	MinTime string `json:"min_time"`
	AvgTime string `json:"avg_time"`
}
