# CW-RailsPathMetrics

A CLI tool for analyzing Rails application logs from AWS CloudWatch Logs. This tool aggregates request metrics by path, providing insights into application performance including request counts and response times.

![Go Version](https://img.shields.io/badge/Go-1.24.5+-blue.svg)

## Features

- **CloudWatch Integration**: Seamlessly fetches logs from AWS CloudWatch Logs with intelligent pagination
- **Rails Log Analysis**: Parses both standard and production format Rails logs with session-based request matching
- **Path Normalization**: Converts dynamic paths to parameterized routes for meaningful aggregation
- **Performance Metrics**: Calculates min/max/average response times and request counts
- **Configurable Exclusions**: Filter out unwanted paths using exact matches, prefixes, or regex patterns
- **JSON Output**: Structured output sorted by request count for easy integration with other tools
- **JST Time Support**: User-friendly time input in JST with automatic UTC conversion for CloudWatch
- **High Performance**: Optimized CloudWatch filter patterns reduce data transfer and processing costs

## Installation

### Prerequisites

- Go 1.24.5 or later
- AWS CLI configured with appropriate permissions
- AWS credentials for CloudWatch Logs access

### Build from Source

```bash
git clone https://github.com/kgrsutos/CW-RailsPathMetrics.git
cd CW-RailsPathMetrics
make build
```

### Using Go Install

TBD

## Usage

### Basic Usage

```bash
./cwrstats analyze \
  --start "2025-07-01T00:00:00" \
  --end "2025-07-01T23:59:59" \
  --log-group "/aws/rails/production-log" \
  --profile myprofile
```

### CLI Options

| Flag | Description | Required | Format |
|------|-------------|----------|---------|
| `--start` | Start time in JST | Yes | `2006-01-02T15:04:05` |
| `--end` | End time in JST | Yes | `2006-01-02T15:04:05` |
| `--log-group` | CloudWatch Logs log group name | Yes | String |
| `--profile` | AWS profile name | Yes | String |

### Output Format

The tool outputs JSON with request metrics sorted by request count (descending):

```json
[
  {
    "path": "/users/:id",
    "count": 1250,
    "max_time_ms": 890,
    "min_time_ms": 45,
    "avg_time_ms": 121
  }
]
```

## Configuration

### Path Exclusions

Configure path exclusions in `config/excluded_paths.yml`:

```yaml
excluded_paths:
  # Exact path match
  - exact: "/health"
  
  # Prefix match
  - prefix: "/rails/active_storage"
  
  # Regex pattern match
  - pattern: "^/api/internal/.*"
```

### AWS Permissions

Ensure your AWS profile has the following IAM permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:FilterLogEvents",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams"
      ],
      "Resource": "arn:aws:logs:*:*:log-group:/aws/rails/*"
    }
  ]
}
```

## Architecture

### Component Overview

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   CLI       │───▶│  CloudWatch  │───▶│  Analysis   │
│   Layer     │    │  Client      │    │  Engine     │
└─────────────┘    └──────────────┘    └─────────────┘
       │                                       │
       ▼                                       ▼
┌─────────────┐                      ┌─────────────┐
│ Time Utils  │                      │ JSON Output │
│ (JST↔UTC)   │                      │             │
└─────────────┘                      └─────────────┘
```

### Analysis Pipeline

1. **Parser**: Extracts structured data from Rails log messages using regex patterns
2. **Normalizer**: Converts dynamic paths to parameterized routes (e.g., `/users/123` → `/users/:id`)
3. **Aggregator**: Matches Started/Completed log pairs by session ID and calculates metrics
4. **Output**: Generates JSON sorted by request count

### Supported Log Formats

**Standard Rails Format:**
```
Started GET "/users/123" for 127.0.0.1 at 2025-01-01 12:00:00 +0900
Completed 200 OK in 150ms (Views: 100.0ms | ActiveRecord: 50.0ms)
```

**Production Format with Log Level:**
```
I, [2025-07-10T17:28:13.282478 #7]  INFO -- : [session-id] Started GET "/users/123" for 127.0.0.1 at 2025-07-10 17:28:13 +0900
I, [2025-07-10T17:28:13.321048 #7]  INFO -- : [session-id] Completed 200 OK in 33ms (Views: 18.3ms | ActiveRecord: 8.0ms)
```

## Development

### Setup

```bash
# Clone and setup
git clone https://github.com/kgrsutos/CW-RailsPathMetrics.git
cd CW-RailsPathMetrics

# Install development tools
make install-tools

# Download dependencies
make deps
```

### Available Commands

```bash
make build          # Build the binary
make test           # Run tests with coverage report
make test-coverage  # Run tests and display coverage percentage
make lint           # Run linters
make clean          # Clean build artifacts
make deps           # Download and tidy dependencies
```

### Testing

Run the full test suite with coverage:

```bash
make test
```

Current test coverage: **84%** overall, **95.8%+** for the analyzer package.

### Architecture Principles

- **Interface-based Design**: All external dependencies use interfaces for easy testing
- **Session-based Matching**: Uses session IDs from log messages for accurate request pairing
- **Clean Architecture**: Clear separation between CLI, CloudWatch client, and analysis layers
- **Graceful Error Handling**: Invalid logs are skipped rather than failing the entire analysis

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Write comprehensive tests for new features
- Document public APIs with Go doc comments
- Use structured logging with `slog`

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI functionality
- Uses [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2) for CloudWatch integration
- Structured logging with Go's standard [slog](https://pkg.go.dev/log/slog) package
