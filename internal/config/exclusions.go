package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// ExclusionRule represents a rule for excluding paths
type ExclusionRule struct {
	Exact   string `yaml:"exact,omitempty"`
	Prefix  string `yaml:"prefix,omitempty"`
	Pattern string `yaml:"pattern,omitempty"`
}

// ExclusionConfig represents the configuration for path exclusions
type ExclusionConfig struct {
	ExcludedPaths []ExclusionRule `yaml:"excluded_paths"`
}

// PathExcluder handles path exclusion logic
type PathExcluder struct {
	config         *ExclusionConfig
	compiledRegexs []*regexp.Regexp
}

// NewPathExcluder creates a new PathExcluder from a config file
func NewPathExcluder(configPath string) (*PathExcluder, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config ExclusionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Validate that each rule has at least one matching criteria
	for i, rule := range config.ExcludedPaths {
		if rule.Exact == "" && rule.Prefix == "" && rule.Pattern == "" {
			return nil, fmt.Errorf("exclusion rule at index %d must specify at least one matching criteria", i)
		}
	}

	excluder := &PathExcluder{
		config:         &config,
		compiledRegexs: make([]*regexp.Regexp, 0),
	}

	// Compile regex patterns
	for _, rule := range config.ExcludedPaths {
		if rule.Pattern != "" {
			// Warn about potentially problematic regex patterns
			if strings.Contains(rule.Pattern, ".*.*") {
				slog.Warn("Regex pattern contains multiple .* which may cause performance issues", "pattern", rule.Pattern)
			}
			if strings.HasPrefix(rule.Pattern, ".*") && !strings.HasPrefix(rule.Pattern, "^") {
				slog.Warn("Regex pattern starts with .* without ^ anchor, consider using prefix match instead", "pattern", rule.Pattern)
			}

			regex, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return nil, fmt.Errorf("failed to compile regex pattern '%s': %w", rule.Pattern, err)
			}
			excluder.compiledRegexs = append(excluder.compiledRegexs, regex)
		} else {
			excluder.compiledRegexs = append(excluder.compiledRegexs, nil)
		}
	}

	return excluder, nil
}

// NewDefaultPathExcluder creates a PathExcluder with default exclusions
func NewDefaultPathExcluder() *PathExcluder {
	config := &ExclusionConfig{
		ExcludedPaths: []ExclusionRule{
			{Prefix: "/rails/active_storage"},
		},
	}

	return &PathExcluder{
		config:         config,
		compiledRegexs: make([]*regexp.Regexp, len(config.ExcludedPaths)),
	}
}

// ShouldExclude checks if a path should be excluded from aggregation
func (pe *PathExcluder) ShouldExclude(path string) bool {
	for i, rule := range pe.config.ExcludedPaths {
		// Exact match
		if rule.Exact != "" && rule.Exact == path {
			return true
		}

		// Prefix match
		if rule.Prefix != "" && strings.HasPrefix(path, rule.Prefix) {
			return true
		}

		// Pattern match
		if rule.Pattern != "" && pe.compiledRegexs[i] != nil {
			if pe.compiledRegexs[i].MatchString(path) {
				return true
			}
		}
	}

	return false
}

// FindConfigPath searches for a configuration file in standard locations
// Returns the path and a boolean indicating whether the file was found
func FindConfigPath() (string, bool) {
	configFilename := "excluded_paths.yml"

	// Search paths in order of preference
	searchPaths := []string{}

	// 1. XDG_CONFIG_HOME/cw-railspathmetrics/excluded_paths.yml
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		searchPaths = append(searchPaths, filepath.Join(xdgConfig, "cw-railspathmetrics", configFilename))
	}

	// 2. HOME/.config/cw-railspathmetrics/excluded_paths.yml
	if home := os.Getenv("HOME"); home != "" {
		searchPaths = append(searchPaths, filepath.Join(home, ".config", "cw-railspathmetrics", configFilename))
		// 3. HOME/.cw-railspathmetrics/excluded_paths.yml
		searchPaths = append(searchPaths, filepath.Join(home, ".cw-railspathmetrics", configFilename))
	}

	slog.Debug("Searching for config file", "paths", searchPaths)

	// Check each path
	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			slog.Info("Found config file", "path", path)
			return path, true
		}
		slog.Debug("Config file not found", "path", path)
	}

	slog.Info("No config file found, using default exclusions")
	return "", false
}

// NewPathExcluderWithSearch creates a PathExcluder by searching for config files in standard locations
// If no config file is found, returns a PathExcluder with default exclusions
func NewPathExcluderWithSearch() (*PathExcluder, error) {
	if configPath, found := FindConfigPath(); found {
		return NewPathExcluder(configPath)
	}

	// No config file found, use default exclusions
	return NewDefaultPathExcluder(), nil
}
