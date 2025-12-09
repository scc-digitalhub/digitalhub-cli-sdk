// SPDX-FileCopyrightText: © 2025 DSLab - Fondazione Bruno Kessler
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os"
	"time"
)

/* ------------ tiny UI helpers for single-line progress ------------ */

type globalProgress struct {
	totalKnown bool
	totalBytes int64
	doneBytes  int64
	spinIdx    int
	lastTick   time.Time
}

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
	// throttling: update ~10 times each seconds to avoid “spamming”
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
