// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"github.com/scc-digitalhub/digitalhub-cli-sdk/sdk/config"

	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

/* ------------ logging helpers (stderr) ------------ */

func infof(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", a...)
}
func warnf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", a...)
}

/* ------------ HTTP (con progress “silenzioso” se possibile) ------------ */

func DownloadHTTPFile(url string, destination string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)

	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	// progress line unica anche senza verbose
	gp := &globalProgress{}
	if resp.ContentLength > 0 {
		gp.totalKnown = true
		gp.totalBytes = resp.ContentLength
	}

	buf := make([]byte, 1024*128) // 128KB
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			gp.add(int64(n))
			gp.render(false)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}
	gp.done()
	return nil
}

/* ------------ S3: file o directory (with continuation token) ------------ */

func DownloadS3FileOrDir(
	s3Client *config.S3Client,
	ctx context.Context,
	parsedPath *ParsedPath,
	localPath string,
	verbose bool,
) error {

	bucket := parsedPath.Host
	// normalizza: rimuovi eventuale leading "/" (alcuni artifact salvano "/xxx/..")
	path := strings.TrimPrefix(parsedPath.Path, "/")

	// Directory?
	if strings.HasSuffix(path, "/") {
		localBase := cleanLocalPath(localPath)

		var totalFiles int
		var totalBytes int64
		var totalsKnown bool

		// Calcolo totals SEMPRE se possibile (serve per la percentuale globale)
		all, err := s3Client.ListFilesAll(ctx, bucket, path)
		if err != nil {
			warnf("Listing failed, proceeding without totals: %v", err)
			infof("Preparing download s3://%s/%s → %s", bucket, path, displayPath(localBase))
			totalsKnown = false
		} else {
			totalFiles = len(all)
			for _, f := range all {
				totalBytes += f.Size
			}
			totalsKnown = totalFiles > 0 && totalBytes > 0
			if verbose {
				infof("Preparing download s3://%s/%s → %s (%d files, %.2f MB)",
					bucket, path, displayPath(localBase), totalFiles, float64(totalBytes)/(1024*1024))
			} else {
				infof("Preparing download s3://%s/%s → %s", bucket, path, displayPath(localBase))
			}
		}

		// Scarica via WalkPrefix (pagination)
		pageSize := int32(1000)
		var idx int

		// Progress globale SOLO quando non-verbose (in verbose mantieni i dettagli per file)
		var gp *globalProgress
		if !verbose {
			gp = &globalProgress{
				totalKnown: totalsKnown,
				totalBytes: totalBytes,
			}
		}

		err = s3Client.WalkPrefix(ctx, bucket, path, pageSize, func(obj s3types.Object) error {
			idx++
			key := aws.ToString(obj.Key)
			relativePath := strings.TrimPrefix(key, path)
			targetPath := filepath.Join(localBase, relativePath)

			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("failed to create local directory: %w", err)
			}

			if verbose {
				if totalFiles > 0 {
					fmt.Fprintf(os.Stderr, "   [%d/%d] %s\n", idx, totalFiles, relativePath)
				} else {
					fmt.Fprintf(os.Stderr, "   [%d] %s\n", idx, relativePath)
				}

				// barra di avanzamento per-file (già presente)
				hook := &config.ProgressHook{
					OnStart: func(k string, total int64) {
						if total > 0 {
							fmt.Fprintf(os.Stderr, "      └─ size: %.2f MB\n", float64(total)/(1024*1024))
						}
					},
					OnProgress: func(k string, written, total int64) {
						if total <= 0 {
							return
						}
						pct := float64(written) / float64(total) * 100
						fmt.Fprintf(os.Stderr, "\r      └─ downloading: %6.2f%%", pct)
					},
					OnDone: func(k string, total int64, took time.Duration) {
						if total > 0 {
							fmt.Fprintf(os.Stderr, "\r      └─ done:        100.00%% in %s\n", took.Truncate(100*time.Millisecond))
						} else {
							fmt.Fprintf(os.Stderr, "      └─ done in %s\n", took.Truncate(100*time.Millisecond))
						}
					},
				}
				if err := s3Client.DownloadFileWithProgress(ctx, bucket, key, targetPath, hook); err != nil {
					return fmt.Errorf("failed to download file: %w", err)
				}
			} else {
				// non-verbose: progress GLOBALE su una riga
				var prevWritten int64
				hook := &config.ProgressHook{
					OnProgress: func(k string, written, total int64) {
						delta := written - prevWritten
						if delta > 0 && gp != nil {
							gp.add(delta)
							gp.render(false)
						}
						prevWritten = written
					},
					OnDone: func(k string, total int64, took time.Duration) {
						// in caso di arrotondamenti, assicurati di contare tutto il file
						if total > prevWritten && gp != nil {
							gp.add(total - prevWritten)
							gp.render(true)
						}
					},
				}
				if err := s3Client.DownloadFileWithProgress(ctx, bucket, key, targetPath, hook); err != nil {
					return fmt.Errorf("failed to download file: %w", err)
				}
			}

			return nil
		})
		if err != nil {
			return err
		}
		if !verbose && gp != nil {
			gp.done()
		}
		return nil
	}

	// Singolo file
	key := path
	if verbose {
		infof("Preparing download s3://%s/%s → %s", bucket, key, displayPath(localPath))
		hook := &config.ProgressHook{
			OnStart: func(k string, total int64) {
				if total > 0 {
					fmt.Fprintf(os.Stderr, "   size: %.2f MB\n", float64(total)/(1024*1024))
				}
			},
			OnProgress: func(k string, written, total int64) {
				if total <= 0 {
					return
				}
				pct := float64(written) / float64(total) * 100
				fmt.Fprintf(os.Stderr, "\r   downloading: %6.2f%%", pct)
			},
			OnDone: func(k string, total int64, took time.Duration) {
				if total > 0 {
					fmt.Fprintf(os.Stderr, "\r   done:        100.00%% in %s\n", took.Truncate(100*time.Millisecond))
				} else {
					fmt.Fprintf(os.Stderr, "   done in %s\n", took.Truncate(100*time.Millisecond))
				}
			},
		}
		if err := s3Client.DownloadFileWithProgress(ctx, bucket, key, localPath, hook); err != nil {
			return fmt.Errorf("S3 download failed: %w", err)
		}
		return nil
	}

	// non-verbose: banner minimo + progress globale su una riga
	infof("Preparing download s3://%s/%s → %s", bucket, key, displayPath(localPath))
	var gp globalProgress
	var prevWritten int64
	hook := &config.ProgressHook{
		OnStart: func(k string, total int64) {
			if total > 0 {
				gp.totalKnown = true
				gp.totalBytes = total
			}
		},
		OnProgress: func(k string, written, total int64) {
			delta := written - prevWritten
			if delta > 0 {
				gp.add(delta)
				gp.render(false)
			}
			prevWritten = written
		},
		OnDone: func(k string, total int64, took time.Duration) {
			if total > prevWritten {
				gp.add(total - prevWritten)
			}
			gp.render(true)
			gp.done()
		},
	}
	if err := s3Client.DownloadFileWithProgress(ctx, bucket, key, localPath, hook); err != nil {
		return fmt.Errorf("S3 download failed: %w", err)
	}
	return nil
}

/* ------------ helpers ------------ */

// Rimuove l’ultimo segmento dal path locale in modo che i file della “cartella” S3
// vengano salvati senza includere il prefisso root.
func cleanLocalPath(path string) string {
	clean := filepath.Clean(path)
	parts := strings.Split(clean, string(os.PathSeparator))
	if len(parts) == 1 {
		return ""
	}
	return filepath.Join(parts[:len(parts)-1]...)
}

// per stampare cartelle vuote come "." invece di stringa vuota
func displayPath(p string) string {
	if p == "" {
		return "."
	}
	return p
}
