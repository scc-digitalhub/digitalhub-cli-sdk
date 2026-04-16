// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

/* ------------ Constants & Resources ------------ */

const (
	RunId = "run_id"
)

var Resources = map[string][]string{
	"artifacts": {"artifact"},
	"dataitems": {"dataitem"},
	"functions": {"function", "fn"},
	"models":    {"model"},
	"projects":  {"project"},
	"runs":      {"run"},
	"workflows": {"workflow"},
	"logs":      {"log"},
}

/* ------------ Types ------------ */

// ParsedPath represents a parsed URI path (S3, HTTP, or local)
type ParsedPath struct {
	Scheme   string
	Host     string
	Path     string
	Filename string
}

// MergeConfig defines how specific fields (arrays of maps) should be merged.
type MergeConfig map[string]string

// globalProgress tracks download/upload progress for single-line display
type globalProgress struct {
	totalKnown bool
	totalBytes int64
	doneBytes  int64
	spinIdx    int
	lastTick   time.Time
}

/* ------------ Logging Helpers ------------ */

func infof(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", a...)
}

func warnf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", a...)
}

/* ------------ Endpoint Translation ------------ */

// TranslateEndpoint translates a resource name to its API endpoint
func TranslateEndpoint(resource string) string {
	for key, val := range Resources {
		if key == resource || contains(val, resource) {
			return key
		}
	}
	// Return the resource as-is if not found (fallback)
	return resource
}

// contains checks if a slice contains a string value
func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

/* ------------ Path Parsing ------------ */

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

/* ------------ Map Utilities ------------ */

// MergeMaps merges two maps giving precedence to map2.
func MergeMaps(map1, map2 map[string]interface{}, cfg MergeConfig) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy map1 into result
	for k, v := range map1 {
		result[k] = v
	}

	// Merge map2 into result
	for k, v2 := range map2 {
		v1, exists := result[k]

		switch {
		// Both are nested maps → merge recursively
		case exists && isMap(v1) && isMap(v2):
			result[k] = MergeMaps(v1.(map[string]interface{}), v2.(map[string]interface{}), cfg)

		// Both are arrays of maps and merge config is defined
		case exists && isSlice(v1) && isSlice(v2) && cfg != nil:
			arr1 := v1.([]interface{})
			arr2 := v2.([]interface{})
			if mergeKey, ok := cfg[k]; ok && looksLikeArrayOfMaps(arr1) && looksLikeArrayOfMaps(arr2) {
				result[k] = mergeArrayOfMapsByKey(arr1, arr2, mergeKey, cfg)
			} else {
				result[k] = v2
			}

		// Default case → overwrite
		default:
			result[k] = v2
		}
	}

	return result
}

func mergeArrayOfMapsByKey(arr1, arr2 []interface{}, key string, cfg MergeConfig) []interface{} {
	index := make(map[interface{}]map[string]interface{})

	// Index elements of arr1 by key
	for _, item := range arr1 {
		if m, ok := item.(map[string]interface{}); ok {
			if id, exists := m[key]; exists {
				index[id] = m
			}
		}
	}

	// Merge elements from arr2 into the index
	for _, item := range arr2 {
		if m, ok := item.(map[string]interface{}); ok {
			if id, exists := m[key]; exists {
				if existing, found := index[id]; found {
					index[id] = MergeMaps(existing, m, cfg)
				} else {
					index[id] = m
				}
			}
		}
	}

	// Convert map back to slice
	result := make([]interface{}, 0, len(index))
	for _, m := range index {
		result = append(result, m)
	}
	return result
}

func looksLikeArrayOfMaps(arr []interface{}) bool {
	for _, item := range arr {
		if _, ok := item.(map[string]interface{}); !ok {
			return false
		}
	}
	return true
}

func isMap(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}

func isSlice(v interface{}) bool {
	_, ok := v.([]interface{})
	return ok
}

/* ------------ UUID Generation ------------ */

// UUIDv4NoDash generates a UUID v4 without dashes
func UUIDv4NoDash() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

/* ------------ Progress Tracking ------------ */

var spinner = []rune{'|', '/', '-', '\\'}

func (gp *globalProgress) add(delta int64) {
	gp.doneBytes += delta
}

func (gp *globalProgress) human(n int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case n >= GB:
		return fmt.Sprintf("%.2f GB", float64(n)/float64(GB))
	case n >= MB:
		return fmt.Sprintf("%.2f MB", float64(n)/float64(MB))
	case n >= KB:
		return fmt.Sprintf("%.2f KB", float64(n)/float64(KB))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func (gp *globalProgress) render(force bool) {
	// throttling: update ~10 times each seconds to avoid "spamming"
	if !force && time.Since(gp.lastTick) < 100*time.Millisecond {
		return
	}
	gp.lastTick = time.Now()

	if gp.totalKnown && gp.totalBytes > 0 {
		pct := float64(gp.doneBytes) / float64(gp.totalBytes) * 100
		if gp.doneBytes > gp.totalBytes {
			gp.doneBytes = gp.totalBytes
			pct = 100
		}
		fmt.Fprintf(os.Stderr, "\rProgress: %6.2f%% (%s / %s)   ",
			pct, gp.human(gp.doneBytes), gp.human(gp.totalBytes))
	} else {
		ch := spinner[gp.spinIdx%len(spinner)]
		gp.spinIdx++
		fmt.Fprintf(os.Stderr, "\rProgress: [%c] %s downloaded   ", ch, gp.human(gp.doneBytes))
	}
}

func (gp *globalProgress) done() {
	gp.render(true)
	fmt.Fprintln(os.Stderr)
}
