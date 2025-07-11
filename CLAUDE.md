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

# Run specific test package
go test -v ./internal/cloudwatch

# Run single test function
go test -v ./internal/analyzer -run TestAggregator_MatchRequestPairs

# Run tests with race detection
go test -race ./...

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

The application follows a clean, layered architecture with clear separation of concerns:

### Layer Structure
- **CLI Layer** (`internal/cli/`): Cobra-based command-line interface handling user input and time parsing
- **CloudWatch Layer** (`internal/cloudwatch/`): AWS SDK wrapper with interface-based design for testability
- **Analysis Layer** (`internal/analyzer/`): Core business logic with four specialized components
- **Models Layer** (`internal/models/`): Well-defined data structures for the entire pipeline
- **Utilities** (`pkg/timeutil/`): Shared utility functions for time processing

### Key Components

#### Analysis Engine (`internal/analyzer/`)
The analysis layer consists of four specialized components that work together:

1. **Parser**: Parses Rails log entries using robust regex patterns
   - Handles both "Started" and "Completed" log types with flexible format support
   - Extracts session IDs, paths, timing, and performance metrics (Views, ActiveRecord)
   - Supports logs with and without log level prefixes

2. **Normalizer**: Converts dynamic paths to parameterized routes for aggregation
   - Normalizes numeric IDs, UUIDs, hex IDs, dates, and order IDs
   - Converts `/users/123` to `/users/:id` and excludes query parameters
   - `/users/123?page=1` becomes `/users/:id` (query parameters are removed for aggregation)
   - Uses sophisticated pattern matching for accurate ID detection

3. **Aggregator**: Matches log pairs and calculates comprehensive metrics
   - **Session ID-based Matching**: Matches Started/Completed logs by session ID (not FIFO)
   - **Path Exclusion**: Filters out configured paths (e.g., `/rails/active_storage`)
   - Calculates average, min, max response times with proper statistical aggregation
   - Aggregates status codes, HTTP methods, and database/view durations

4. **Analyzer**: Coordinates the entire analysis process
   - Orchestrates parsing, normalization, and aggregation in proper sequence
   - Provides structured JSON output sorted by request count (descending)
   - Handles invalid log entries gracefully

#### CloudWatch Integration
- **Interface-based Design**: Uses `CloudWatchLogsAPI` interface for easy mocking and testing
- **Pagination Support**: Implements `FilterLogEventsWithPagination` for large log volumes
- **Profile-based Authentication**: Supports AWS profile configuration
- **Proper Time Handling**: Converts CloudWatch timestamps (milliseconds) to Go time.Time

#### Data Flow
1. CLI parses user input and validates time ranges (JST → UTC conversion)
2. CloudWatch client fetches log events with pagination support
3. Parser extracts structured data from Rails log entries
4. Normalizer standardizes request paths for aggregation (removes query parameters)
5. Aggregator filters paths using exclusion rules and matches request pairs
6. Aggregator calculates metrics and statistics
7. Analyzer outputs structured JSON results sorted by request count

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
The application supports multiple Rails log formats:

**Standard Format:**
```
Started GET "/users/123" for 127.0.0.1 at 2023-01-01 12:00:00 +0900
Completed 200 OK in 150ms (Views: 100.0ms | ActiveRecord: 50.0ms)
```

**Production Format with Log Level Prefix:**
```
I, [2025-07-10T17:28:13.282478 #7]  INFO -- : [session-id] Started GET "/users/123" for 127.0.0.1 at 2025-07-10 17:28:13 +0900
I, [2025-07-10T17:28:13.321048 #7]  INFO -- : [session-id] Completed 200 OK in 33ms (Views: 18.3ms | ActiveRecord: 8.0ms | Allocations: 4970)
```

**Key Features:**
- **Session ID Matching**: SessionID is extracted from both Started and Completed logs for proper pairing
- **Flexible Parsing**: Handles logs with or without log level prefixes automatically
- **Performance Metrics**: Extracts Views duration, ActiveRecord duration, and additional metrics like Allocations
- **Graceful Handling**: Logs without SessionID are processed but cannot be matched for timing metrics

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
- `internal/config/`: Path exclusion configuration management
- `internal/models/`: Data type definitions
- `config/excluded_paths.yml`: Path exclusion configuration file
- `pkg/timeutil/`: Time processing utilities

## Dependencies

- **CLI**: github.com/spf13/cobra
- **AWS SDK**: github.com/aws/aws-sdk-go-v2
- **Testing**: github.com/stretchr/testify
- **Logging**: Standard library slog
- **Linting**: golangci-lint v1.61.0
- **YAML**: gopkg.in/yaml.v3 (for path exclusion configuration)

## Time Handling

All times are handled in JST (Asia/Tokyo) for input parsing but converted to UTC for internal processing and AWS API calls. CloudWatch timestamps are in milliseconds since epoch.

## Implementation Notes

### Current Implementation Status
- ✅ **Analysis Engine**: Fully implemented with comprehensive test coverage (93.2%)
- ✅ **CloudWatch Integration**: Complete with pagination and authentication
- ✅ **CLI Interface**: Functional command-line interface with proper parameter validation
- ✅ **Full Integration**: Complete end-to-end pipeline from CloudWatch logs to JSON output
- ✅ **Path Exclusion**: Configurable path filtering with YAML configuration
- ✅ **JSON Output**: Sorted by request count in descending order

The application is production-ready with SessionID-based matching, path normalization, metrics aggregation, and configurable path exclusion.

### Key Design Patterns
- **Interface-based Design**: All major components use interfaces for testability (CloudWatchLogsAPI)
- **Dependency Injection**: Components accept dependencies via constructors
- **Session-based Matching**: Aggregator matches Started/Completed logs by session ID, not FIFO order
- **Graceful Error Handling**: Invalid log entries are skipped rather than causing failures
- **Structured Logging**: Uses slog with JSON formatting for consistent log output

### Performance Considerations
- **Pagination**: CloudWatch client handles large log volumes efficiently
- **Memory Management**: Processes logs in streams rather than loading everything into memory
- **Efficient Matching**: Session-based matching algorithm avoids O(n²) complexity

## Path Exclusion Configuration

The application supports excluding specific paths from analysis using a YAML configuration file at `config/excluded_paths.yml`:

```yaml
excluded_paths:
  # Rails Active Storage paths (file uploads, downloads, etc.)
  - prefix: "/rails/active_storage"
  
  # Additional examples:
  # - exact: "/health"                # Health check endpoint
  # - prefix: "/assets"               # Static assets  
  # - pattern: "^/api/internal/.*"    # Internal API endpoints
```

**Exclusion Rule Types:**
- `exact`: Complete path match
- `prefix`: Match all paths starting with the specified prefix
- `pattern`: Regular expression pattern matching

Default exclusions include Rails Active Storage paths to avoid skewing metrics with file upload/download traffic.