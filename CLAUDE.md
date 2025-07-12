# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CW-RailsPathMetrics is a CLI application that analyzes Rails application logs from AWS CloudWatch Logs and generates aggregated metrics by request path. The application fetches CloudWatch logs, parses Rails log entries, and outputs JSON statistics including request counts, response times, and status codes.

## Commands

### Development Commands
```bash
# Build the application
make build

# Run all tests with coverage report
make test

# Run tests and display coverage percentage
make test-coverage

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

# Install development tools (golangci-lint)
make install-tools

# Download and tidy dependencies
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

**Note**: All times are in JST (Asia/Tokyo) timezone.

## Architecture

The application follows a clean, layered architecture with clear separation of concerns:

### Core Components

#### CLI Layer (`internal/cli/`)
- Cobra-based command-line interface
- Handles time parsing (JST to UTC conversion)
- Currently supports only the `analyze` command

#### CloudWatch Integration (`internal/cloudwatch/`)
- Interface-based design with `CloudWatchLogsAPI` for testability
- Implements pagination via `FilterLogEventsWithPagination`
- AWS profile-based authentication using AWS SDK v2
- Converts CloudWatch timestamps (milliseconds) to Go time.Time
- **Filter Pattern Optimization**: Uses `?Started ?Completed` pattern to reduce data transfer and costs by filtering at CloudWatch level

#### Analysis Engine (`internal/analyzer/`)
The analysis layer consists of four specialized components:

1. **Parser** (`parser.go`):
   - Parses Rails log entries using regex patterns
   - Handles both "Started" and "Completed" log types
   - Supports production format with log level prefixes
   - Extracts session IDs for request matching

2. **Normalizer** (`normalizer.go`):
   - Converts dynamic paths to parameterized routes
   - Normalizes: numeric IDs, UUIDs, hex IDs, dates, order IDs
   - Removes query parameters for aggregation
   - Example: `/users/123?page=1` → `/users/:id`

3. **Aggregator** (`aggregator.go`):
   - Matches Started/Completed logs by session ID (not FIFO)
   - Filters paths using exclusion rules
   - Calculates min/max/average response times
   - Groups by HTTP method and status code

4. **Analyzer** (`analyzer.go`):
   - Orchestrates the analysis pipeline
   - Outputs JSON sorted by request count (descending)

### Data Flow
1. CLI validates input and converts JST times to UTC
2. CloudWatch client fetches logs with pagination
3. Parser extracts structured data from log messages
4. Normalizer standardizes paths for aggregation
5. Aggregator matches pairs and calculates metrics
6. Results output as JSON to stdout

## Rails Log Format Support

**Standard Format:**
```
Started GET "/users/123" for 127.0.0.1 at 2023-01-01 12:00:00 +0900
Completed 200 OK in 150ms (Views: 100.0ms | ActiveRecord: 50.0ms)
```

**Production Format with Log Level:**
```
I, [2025-07-10T17:28:13.282478 #7]  INFO -- : [session-id] Started GET "/users/123" for 127.0.0.1 at 2025-07-10 17:28:13 +0900
I, [2025-07-10T17:28:13.321048 #7]  INFO -- : [session-id] Completed 200 OK in 33ms (Views: 18.3ms | ActiveRecord: 8.0ms | Allocations: 4970)
```

## Path Exclusion Configuration

Default exclusions are hardcoded in `config/excluded_paths.yml`:
```yaml
excluded_paths:
  - prefix: "/rails/active_storage"
```

The codebase supports three exclusion types:
- `exact`: Complete path match
- `prefix`: Match paths starting with prefix
- `pattern`: Regular expression matching


## Testing

Current coverage: ~84% overall, 95.8%+ for analyzer package

Key test patterns:
- Mock interfaces for AWS SDK (`MockCloudWatchLogsAPI`)
- Table-driven tests throughout
- Edge cases for log parsing and path normalization
- Session ID matching validation
- Race detection enabled in test runs

## Implementation Status

- ✅ Core analysis engine with session-based matching
- ✅ CloudWatch integration with pagination
- ✅ Path normalization and exclusion
- ✅ JSON output sorted by request count
- ❌ Multiple output formats
- ❌ Real-time log streaming

## Key Design Decisions

1. **Session-based Matching**: Uses session IDs from log messages to match Started/Completed pairs accurately (not chronological order)
2. **Interface-based Testing**: All external dependencies use interfaces for easy mocking
3. **JST Time Handling**: User inputs in JST, internal processing in UTC
4. **Path Normalization**: Query parameters stripped, dynamic segments replaced with placeholders
5. **Graceful Degradation**: Invalid logs are skipped rather than failing the entire analysis
6. **Pipeline Architecture**: Clean separation of parsing → normalization → aggregation → output

## Important Notes

- **AWS SDK Version**: Uses AWS SDK v2 (not v1) - ensure compatibility when making changes
- **Time Zone Handling**: All user input times are treated as JST and converted to UTC internally
- **Filter Pattern Syntax**: CloudWatch filter patterns use `?Started ?Completed` syntax for unstructured Rails logs
- **Session ID Extraction**: Session IDs are extracted from log messages using regex patterns, supporting both standard and production log formats
- **Error Handling**: Application uses structured logging (slog) and handles AWS API errors gracefully