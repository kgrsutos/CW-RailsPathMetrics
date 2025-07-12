package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
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
		return nil, err
	}

	var config ExclusionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	excluder := &PathExcluder{
		config:         &config,
		compiledRegexs: make([]*regexp.Regexp, 0),
	}

	// Compile regex patterns
	for _, rule := range config.ExcludedPaths {
		if rule.Pattern != "" {
			regex, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return nil, err
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
	
	// Check each path
	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	
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
