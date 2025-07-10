package analyzer

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kgrsutos/cw-railspathmetrics/internal/models"
)

var (
	// Regular expressions for parsing Rails logs
	startedLogRegex   = regexp.MustCompile(`^Started\s+(\w+)\s+"([^"]+)"\s+for\s+[\d.]+\s+at\s+(.+)$`)
	completedLogRegex = regexp.MustCompile(`^Completed\s+(\d+)\s+([^i]+)\s+in\s+(\d+)ms`)
	viewDurationRegex = regexp.MustCompile(`Views:\s+([\d.]+)ms`)
	dbDurationRegex   = regexp.MustCompile(`ActiveRecord:\s+([\d.]+)ms`)
	sessionIDRegex    = regexp.MustCompile(`\[([^\]]+)\]$`)
)

// Parser handles parsing of Rails log entries
type Parser struct{}

// NewParser creates a new Parser instance
func NewParser() *Parser {
	return &Parser{}
}

// ParseLogEntry parses a single log line and returns a LogEntry
func (p *Parser) ParseLogEntry(logLine string) (*models.LogEntry, error) {
	logLine = strings.TrimSpace(logLine)
	if logLine == "" {
		return nil, errors.New("empty log line")
	}

	if p.isStartedLog(logLine) {
		return p.parseStartedLog(logLine)
	}

	if p.isCompletedLog(logLine) {
		return p.parseCompletedLog(logLine)
	}

	return nil, fmt.Errorf("unrecognized log format: %s", logLine)
}

// isStartedLog checks if the log line is a Started log
func (p *Parser) isStartedLog(logLine string) bool {
	return startedLogRegex.MatchString(logLine)
}

// isCompletedLog checks if the log line is a Completed log
func (p *Parser) isCompletedLog(logLine string) bool {
	return completedLogRegex.MatchString(logLine)
}

// parseStartedLog parses a Started log entry
func (p *Parser) parseStartedLog(logLine string) (*models.LogEntry, error) {
	matches := startedLogRegex.FindStringSubmatch(logLine)
	if len(matches) != 4 {
		return nil, fmt.Errorf("invalid Started log format: %s", logLine)
	}

	// Parse timestamp
	timestamp, err := p.parseTimestamp(matches[3])
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return &models.LogEntry{
		Type:      "Started",
		Method:    matches[1],
		Path:      matches[2],
		Timestamp: timestamp,
	}, nil
}

// parseCompletedLog parses a Completed log entry
func (p *Parser) parseCompletedLog(logLine string) (*models.LogEntry, error) {
	matches := completedLogRegex.FindStringSubmatch(logLine)
	if len(matches) != 4 {
		return nil, fmt.Errorf("invalid Completed log format: %s", logLine)
	}

	statusCode, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid status code: %w", err)
	}

	duration, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}

	entry := &models.LogEntry{
		Type:       "Completed",
		StatusCode: statusCode,
		StatusText: strings.TrimSpace(matches[2]),
		Duration:   duration,
		SessionID:  p.extractSessionID(logLine),
	}

	// Extract view duration if present
	if viewMatches := viewDurationRegex.FindStringSubmatch(logLine); len(viewMatches) > 1 {
		if viewDuration, err := strconv.ParseFloat(viewMatches[1], 64); err == nil {
			entry.ViewDuration = viewDuration
		}
	}

	// Extract DB duration if present
	if dbMatches := dbDurationRegex.FindStringSubmatch(logLine); len(dbMatches) > 1 {
		if dbDuration, err := strconv.ParseFloat(dbMatches[1], 64); err == nil {
			entry.DBDuration = dbDuration
		}
	}

	return entry, nil
}

// extractSessionID extracts session ID from log line
func (p *Parser) extractSessionID(logLine string) string {
	matches := sessionIDRegex.FindStringSubmatch(logLine)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// parseTimestamp parses timestamp from Rails log format
func (p *Parser) parseTimestamp(timestampStr string) (time.Time, error) {
	// Rails log timestamp format: "2023-01-01 12:00:00 +0900"
	const layout = "2006-01-02 15:04:05 -0700"
	return time.Parse(layout, timestampStr)
}
