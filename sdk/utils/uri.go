package utils

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type ParsedPath struct {
	Scheme   string
	Host     string
	Path     string
	Filename string
}

// ParsePath parses any kind of path: S3, HTTP, local (absolute or relative)
func ParsePath(input string) (*ParsedPath, error) {
	// Try parsing as URI
	parsed, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path: %w", err)
	}

	result := &ParsedPath{}

	// If there's a scheme (e.g. s3, https), treat it as URI
	if parsed.Scheme != "" {
		result.Scheme = parsed.Scheme
		result.Host = parsed.Host
		result.Path = strings.TrimPrefix(parsed.Path, "/")
		result.Filename = filepath.Base(parsed.Path)
		return result, nil
	}

	// Else, it's a local path
	result.Scheme = "file"
	result.Host = ""
	result.Path = input
	result.Filename = filepath.Base(input)

	return result, nil
}
