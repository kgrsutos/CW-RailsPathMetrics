# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CW-RailsPathMetrics is a CLI application that analyzes Rails application logs from AWS CloudWatch Logs and generates aggregated metrics by request path. The application fetches CloudWatch logs, parses Rails log entries, and outputs JSON statistics including request counts, response times, and status codes.

## Commands

### Development Commands
```bash
# Build the application
make build

# Run all tests with coverage report and HTML output
make test

# Run tests and display coverage percentage only
make test-coverage

# Run specific test package
go test -v ./internal/cloudwatch

# Run single test function
go test -v ./internal/analyzer -run TestAggregator_MatchRequestPairs

# Run integration tests specifically
go test -v ./internal/integration

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
# Analyze CloudWatch logs with default configuration
./cwrstats analyze \
  --start "2025-07-01T00:00:00" \
  --end "2025-07-01T23:59:59" \
  --log-group "/aws/rails/production-log" \
  --profile myprofile

# Use custom configuration file
./cwrstats analyze \
  --config /path/to/custom/excluded_paths.yml \
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
- Optional `--config` flag for custom exclusion configuration files

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
   - Supports custom configuration via `NewAnalyzerWithConfig()`

#### Configuration System (`internal/config/`)
- **Path Exclusion**: Supports exact, prefix, and regex pattern matching
- **Auto-discovery**: Searches standard configuration locations following XDG Base Directory specification
- **Fallback**: Uses hardcoded defaults when no configuration file is found
- **File Locations** (in order of preference):
  1. `$XDG_CONFIG_HOME/cw-railspathmetrics/excluded_paths.yml`
  2. `$HOME/.config/cw-railspathmetrics/excluded_paths.yml`
  3. `$HOME/.cw-railspathmetrics/excluded_paths.yml`

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
Started GET "/users/123" for 127.0.0.1 at 2023-01-01 12:00:00 +0900 [session-id]
Completed 200 OK in 150ms (Views: 100.0ms | ActiveRecord: 50.0ms) [session-id]
```

**Production Format with Log Level:**
```
I, [2025-07-10T17:28:13.282478 #7]  INFO -- : [session-id] Started GET "/users/123" for 127.0.0.1 at 2025-07-10 17:28:13 +0900
I, [2025-07-10T17:28:13.321048 #7]  INFO -- : [session-id] Completed 200 OK in 33ms (Views: 18.3ms | ActiveRecord: 8.0ms | Allocations: 4970)
```

**Important**: Session IDs are crucial for matching Started/Completed log pairs. Both formats are supported with session IDs appearing in square brackets.

## Path Exclusion Configuration

### Configuration File Format
```yaml
excluded_paths:
  - exact: "/health"                    # Exact path match
  - prefix: "/rails/active_storage"     # Prefix match
  - pattern: "^/api/internal/.*"        # Regex pattern match
```

### Configuration Loading Strategy
The application uses a layered configuration approach:

1. **CLI Flag**: `--config /path/to/custom.yml` (highest priority)
2. **Auto-discovery**: Searches standard locations in order:
   - `$XDG_CONFIG_HOME/cw-railspathmetrics/excluded_paths.yml`
   - `$HOME/.config/cw-railspathmetrics/excluded_paths.yml`
   - `$HOME/.cw-railspathmetrics/excluded_paths.yml`
3. **Default**: Hardcoded exclusion for `/rails/active_storage` prefix (fallback)

### Go Install Distribution
For `go install` distribution, users can create configuration files in standard locations:

```bash
# Create config directory
mkdir -p ~/.config/cw-railspathmetrics

# Create configuration file
cat > ~/.config/cw-railspathmetrics/excluded_paths.yml << EOF
excluded_paths:
  - exact: "/health"
  - prefix: "/assets"
  - pattern: "^/api/internal/.*"
EOF
```

## Testing

Current coverage: ~84% overall, 95.8%+ for analyzer package

### Test Structure
- **Unit Tests**: Each component (`parser`, `normalizer`, `aggregator`) has comprehensive unit tests
- **Integration Tests**: Full workflow tests in `internal/integration/` that mock AWS CloudWatch and test complete data flow
- **CLI Tests**: Command-line interface and time conversion tests

### Key Test Patterns
- Mock interfaces for AWS SDK (`MockCloudWatchLogsAPI`)
- Table-driven tests throughout
- Edge cases for log parsing and path normalization
- Session ID matching validation
- Race detection enabled in test runs

### Integration Test Coverage
The integration tests (`internal/integration/`) provide comprehensive coverage of:
- Complete CloudWatch → Parser → Normalizer → Aggregator workflow
- Session-based log matching with interleaved sessions
- Path exclusion functionality
- CloudWatch pagination handling
- Error handling scenarios
- JST to UTC time conversion

## Implementation Status

- ✅ Core analysis engine with session-based matching
- ✅ CloudWatch integration with pagination
- ✅ Path normalization and exclusion with configurable rules
- ✅ JSON output sorted by request count
- ✅ Configuration system with auto-discovery and XDG Base Directory support
- ✅ CLI --config flag for custom configuration files
- ✅ Comprehensive integration tests including configuration scenarios
- ✅ Go install distribution ready
- ❌ Multiple output formats
- ❌ Real-time log streaming

## Key Design Decisions

1. **Session-based Matching**: Uses session IDs from log messages to match Started/Completed pairs accurately (not chronological order)
2. **Interface-based Testing**: All external dependencies use interfaces for easy mocking
3. **JST Time Handling**: User inputs in JST, internal processing in UTC
4. **Path Normalization**: Query parameters stripped, dynamic segments replaced with placeholders
5. **Graceful Degradation**: Invalid logs are skipped rather than failing the entire analysis
6. **Pipeline Architecture**: Clean separation of parsing → normalization → aggregation → output
7. **Configuration Strategy**: XDG Base Directory compliance with fallback to hardcoded defaults
8. **Go Install Ready**: Configuration auto-discovery enables distribution via `go install`

## Important Notes

- **AWS SDK Version**: Uses AWS SDK v2 (not v1) - ensure compatibility when making changes
- **Time Zone Handling**: All user input times are treated as JST and converted to UTC internally
- **Filter Pattern Syntax**: CloudWatch filter patterns use `?Started ?Completed` syntax for unstructured Rails logs
- **Session ID Extraction**: Session IDs are extracted from log messages using regex patterns, supporting both standard and production log formats
- **Error Handling**: Application uses structured logging (slog) and handles AWS API errors gracefully
- **Module Name**: `github.com/kgrsutos/cw-railspathmetrics` - use this for imports and go.mod references

## Development Workflow

### Adding New Features
1. Write unit tests first (table-driven pattern preferred)
2. Implement feature with interface-based design for external dependencies
3. Add integration tests if the feature affects the complete workflow
4. Run `make test` to ensure all tests pass
5. Run `make lint` to ensure code quality

### Testing New Log Formats
When adding support for new Rails log formats:
1. Add test cases to `internal/analyzer/parser_test.go`
2. Update regex patterns in `internal/analyzer/parser.go`
3. Verify integration tests still pass
4. Update documentation in CLAUDE.md and README.md

### Adding Configuration Features
When extending the configuration system:
1. Update `internal/config/exclusions.go` for new config options
2. Add corresponding tests to `internal/config/exclusions_test.go`
3. Update `NewAnalyzerWithConfig()` in `internal/analyzer/analyzer.go` if needed
4. Test auto-discovery and fallback behavior
5. Update CLI help text and documentation

### Debugging Issues
- Use `go test -v` with specific package/function for detailed test output
- Integration tests provide end-to-end debugging capability
- Check `coverage.html` for test coverage gaps
- Use race detection: `go test -race ./...`
- Test configuration loading with: `CLOUDWATCH_LOG_GROUP=test go run ./cmd/cwrstats --help`