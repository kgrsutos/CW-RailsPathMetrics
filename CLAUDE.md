# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CW-RailsPathMetrics is a CLI application that analyzes Rails application logs from AWS CloudWatch Logs and generates aggregated metrics by request path. The application fetches CloudWatch logs, parses Rails log entries, and outputs JSON statistics including request counts, response times, and status codes.

## Commands

### Development Commands
```bash
# Build the application
make build

# Run all tests with coverage
make test

# Run specific test
go test -v ./internal/cloudwatch

# Run linter
make lint

# Clean build artifacts
make clean

# Install development tools
make install-tools

# Download dependencies
make deps
```

### Application Usage
```bash
# Analyze CloudWatch logs
./cwrstats analyze \
  --start "2025-07-01T00:00:00" \
  --end "2025-07-01T23:59:59" \
  --log-group "/aws/rails/production-log" \
  --profile myprofile
```

## Architecture

The application follows a layered architecture:

- **CLI Layer** (`internal/cli/`): Cobra-based command-line interface
- **CloudWatch Layer** (`internal/cloudwatch/`): AWS SDK wrapper for CloudWatch Logs API
- **Analysis Layer** (`internal/analyzer/`): Log parsing and aggregation logic
- **Models Layer** (`internal/models/`): Data structures and types

### Key Components

1. **CloudWatch Client**: Handles AWS authentication and log retrieval with pagination support
2. **Log Parser**: Parses Rails log entries (Started/Completed) and extracts session IDs, paths, and timing
3. **Path Normalizer**: Converts parameterized paths like `/users/123` to `/users/:id`
4. **Aggregator**: Matches Started/Completed log pairs and calculates metrics

## Development Guidelines

### Testing Strategy
- Use TDD approach with comprehensive test coverage
- Use testify for assertions and mocking
- Test edge cases and error conditions
- Target 80%+ overall coverage, 90%+ for core logic

### Code Style
- Follow Go conventions and use gofmt
- Use structured logging with slog
- Implement proper error handling with wrapped errors
- Use interfaces for testability (e.g., CloudWatchLogsAPI)

### Rails Log Format
The application expects Rails logs in this format:
```
Started GET "/users/123" for 127.0.0.1 at 2023-01-01 12:00:00 +0900
Completed 200 OK in 150ms (Views: 100.0ms | ActiveRecord: 50.0ms)
```

### AWS Integration
- Uses AWS SDK v2 with profile-based authentication
- Implements pagination for large log volumes
- Handles CloudWatch API rate limits appropriately
- Converts CloudWatch timestamps (milliseconds) to Go time.Time

## File Organization

- `cmd/cwrstats/main.go`: Application entry point
- `internal/cli/`: Command-line interface implementation
- `internal/cloudwatch/`: AWS CloudWatch integration
- `internal/analyzer/`: Log analysis and aggregation
- `internal/models/`: Data type definitions
- `pkg/timeutil/`: Time processing utilities

## Dependencies

- **CLI**: github.com/spf13/cobra
- **AWS SDK**: github.com/aws/aws-sdk-go-v2
- **Testing**: github.com/stretchr/testify
- **Logging**: Standard library slog
- **Linting**: golangci-lint v1.61.0

## Time Handling

All times are handled in JST (Asia/Tokyo) for input parsing but converted to UTC for internal processing and AWS API calls. CloudWatch timestamps are in milliseconds since epoch.