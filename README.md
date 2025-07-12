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

```bash
go install github.com/kgrsutos/cw-railspathmetrics/cmd/cwrstats@latest
```

After installation, the binary will be available in your `$GOPATH/bin` directory.

## Usage

### Basic Usage

```bash
# Using default configuration
./cwrstats analyze \
  --start "2025-07-01T00:00:00" \
  --end "2025-07-01T23:59:59" \
  --log-group "/aws/rails/production-log" \
  --profile myprofile

# Using custom configuration file
./cwrstats analyze \
  --config /path/to/excluded_paths.yml \
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
| `--config` | Path to custom exclusion configuration file | No | String |

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

The application supports configurable path exclusions with automatic configuration file discovery.

#### Configuration File Format

```yaml
excluded_paths:
  # Exact path match
  - exact: "/health"
  
  # Prefix match
  - prefix: "/rails/active_storage"
  
  # Regex pattern match
  - pattern: "^/api/internal/.*"
```

#### Configuration File Locations

The application searches for configuration files in the following locations (in order of preference):

1. **Custom path** (via `--config` flag): `/path/to/custom/excluded_paths.yml`
2. **XDG config directory**: `$XDG_CONFIG_HOME/cw-railspathmetrics/excluded_paths.yml`
3. **User config directory**: `$HOME/.config/cw-railspathmetrics/excluded_paths.yml`
4. **User app directory**: `$HOME/.cw-railspathmetrics/excluded_paths.yml`
5. **Default fallback**: Built-in exclusion for `/rails/active_storage` prefix

#### Setting Up Configuration for Go Install

After installing via `go install`, create a configuration file:

```bash
# Create config directory
mkdir -p ~/.config/cw-railspathmetrics

# Create configuration file
cat > ~/.config/cw-railspathmetrics/excluded_paths.yml << EOF
excluded_paths:
  - exact: "/health"
  - exact: "/ping"
  - prefix: "/rails/active_storage"
  - prefix: "/assets"
  - pattern: "^/api/internal/.*"
EOF
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
│ Config      │                      │ JSON Output │
│ System      │                      │             │
└─────────────┘                      └─────────────┘
       │
       ▼
┌─────────────┐
│ Time Utils  │
│ (JST↔UTC)   │
└─────────────┘
```

### Analysis Pipeline

1. **Configuration**: Loads exclusion rules from config files or uses defaults
2. **Parser**: Extracts structured data from Rails log messages using regex patterns
3. **Normalizer**: Converts dynamic paths to parameterized routes (e.g., `/users/123` → `/users/:id`)
4. **Aggregator**: Matches Started/Completed log pairs by session ID, applies exclusion filters, and calculates metrics
5. **Output**: Generates JSON sorted by request count

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

### Architecture Principles

- **Interface-based Design**: All external dependencies use interfaces for easy testing
- **Session-based Matching**: Uses session IDs from log messages for accurate request pairing
- **Clean Architecture**: Clear separation between CLI, CloudWatch client, configuration, and analysis layers
- **Configuration Auto-discovery**: Follows XDG Base Directory specification with graceful fallbacks
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
