// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package transfer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *TransferService) Download(ctx context.Context, endpoint string, req DownloadRequest) ([]DownloadInfo, error) {
	if req.Resource != "projects" && req.Project == "" {
		return nil, errors.New("project is mandatory for non-project resources")
	}
	if req.ID == "" && req.Name == "" {
		return nil, errors.New("you must specify id or name")
	}

	params := map[string]string{}
	id := req.ID
	if id == "" {
		params["name"] = req.Name
		params["versions"] = "latest"
	}

	url := s.http.BuildURL(req.Project, endpoint, id, params)
	body, _, err := s.http.Do(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	paths, err := extractPaths(body)
	if err != nil {
		return nil, err
	}

	var out []DownloadInfo
	for _, p := range paths {
		pp, err := ParsePath(p)
		if err != nil {
			continue
		}
		target, createdDir, err := chooseLocalTarget(req.Destination, pp.Filename)
		if err != nil {
			continue
		}
		_ = createdDir

		switch pp.Scheme {
		case "s3":
			key := strings.TrimPrefix(pp.Path, "/")
			if strings.HasSuffix(key, "/") {
				// Directory (paginated): on error, do not fail everything; skip it.
				if derr := DownloadS3FileOrDir(s.s3, ctx, pp, target, req.Verbose); derr != nil {
					// Skip the directory (log at the CLI level if desired).
					continue
				}
				// reporting
				files, lerr := s.s3.ListFilesAll(ctx, pp.Host, key)
				if lerr != nil {
					// Warn and skip reporting, but do not fail.
					continue
				}
				base := dirBaseForLocalTarget(target)
				for _, f := range files {
					local := filepath.Join(base, strings.TrimPrefix(f.Path, key))
					if st, err := os.Stat(local); err == nil && !st.IsDir() {
						out = append(out, DownloadInfo{
							Filename: filepath.Base(local),
							Size:     st.Size(),
							Path:     local,
						})
					}
				}
			} else {
				// Single file: on error, do not fail; skip it.
				if ferr := DownloadS3FileOrDir(s.s3, ctx, pp, target, req.Verbose); ferr != nil {
					continue
				}
				if st, err := os.Stat(target); err == nil && !st.IsDir() {
					out = append(out, DownloadInfo{
						Filename: filepath.Base(target),
						Size:     st.Size(),
						Path:     target,
					})
				}
			}

		case "http", "https":
			// On HTTP error, skip (as in the original).
			if herr := DownloadHTTPFile(pp.Path, target); herr != nil {
				continue
			}
			if st, err := os.Stat(target); err == nil && !st.IsDir() {
				out = append(out, DownloadInfo{
					Filename: filepath.Base(target),
					Size:     st.Size(),
					Path:     target,
				})
			}

		default:
			// Unsupported; skip (as in the original).
			continue
		}
	}
	return out, nil
}

// --- helpers ---

// chooseLocalTarget mirrors the original behavior:
// - if dst is empty, use filename in the current working directory
// - if dst exists and is a directory, use dst/filename
// - if dst exists and is a file, use dst
// - if dst does not exist, create directory dst and use dst/filename
func chooseLocalTarget(dst, filename string) (target string, createdDir bool, err error) {
	if dst == "" {
		return filename, false, nil
	}
	info, statErr := os.Stat(dst)
	if statErr == nil {
		if info.IsDir() {
			return filepath.Join(dst, filename), false, nil
		}
		return dst, false, nil // file esistente
	}
	if os.IsNotExist(statErr) {
		// Original behavior: create directory and use dst/filename.
		if mkErr := os.MkdirAll(dst, 0o755); mkErr != nil {
			return "", false, mkErr
		}
		return filepath.Join(dst, filename), true, nil
	}
	// For any other Stat error, propagate it (the caller skips it).
	return "", false, statErr
}

func dirBaseForLocalTarget(localPath string) string {
	clean := filepath.Clean(localPath)
	parent := filepath.Dir(clean)
	if parent == "." || parent == string(os.PathSeparator) {
		return ""
	}
	return parent
}

func extractPaths(body []byte) ([]string, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}
	// If an ID is specified, return a single object.
	if _, has := raw["content"]; !has {
		if spec, ok := raw["spec"].(map[string]interface{}); ok {
			if path, _ := spec["path"].(string); path != "" {
				return []string{path}, nil
			}
		}
		return nil, fmt.Errorf("missing spec.path")
	}
	// content[]
	content, ok := raw["content"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid content")
	}
	var paths []string
	for _, it := range content {
		if m, ok := it.(map[string]interface{}); ok {
			if spec, ok2 := m["spec"].(map[string]interface{}); ok2 {
				if p, _ := spec["path"].(string); p != "" {
					paths = append(paths, p)
				}
			}
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no paths in content")
	}
	return paths, nil
}
