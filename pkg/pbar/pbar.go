// Copyright (c) 2025 Stefano Scafiti
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
package pbar

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ostafen/digler/pkg/util/format"
)

const MinRefreshRate = time.Millisecond * 500

// ProgressBarState holds all the data needed to render the progress bar
type ProgressBarState struct {
	TotalBytes         int64
	ProcessedBytes     int64
	FilesFound         int
	StartTime          time.Time
	LastUpdateTime     time.Time
	LastProcessedBytes int64
}

// NewProgressBarState initializes a new ProgressBarState
func NewProgressBarState(totalBytes int64) *ProgressBarState {
	return &ProgressBarState{
		TotalBytes:         totalBytes,
		ProcessedBytes:     0,
		FilesFound:         0,
		StartTime:          time.Now(),
		LastUpdateTime:     time.Unix(0, 0),
		LastProcessedBytes: 0,
	}
}

// Render updates and prints the progress bar line
func (pbs *ProgressBarState) Render(force bool) {
	if !force && (pbs.LastUpdateTime.IsZero() || time.Since(pbs.LastUpdateTime) < MinRefreshRate) {
		return
	}

	percentage := float64(pbs.ProcessedBytes) / float64(pbs.TotalBytes) * 100

	barLength := 20
	filledLen := int(float64(barLength) * percentage / 100)
	var bar string
	if filledLen == barLength {
		bar = strings.Repeat("=", barLength)
	} else {
		bar = strings.Repeat("=", filledLen) + ">" + strings.Repeat(" ", barLength-filledLen-1)
	}

	//elapsedTime := time.Since(pbs.StartTime)
	currentSpeedBytesPerSec := float64(pbs.ProcessedBytes-pbs.LastProcessedBytes) / time.Since(pbs.LastUpdateTime).Seconds()
	currentSpeedMBps := currentSpeedBytesPerSec / (1024 * 1024)

	var etaStr string
	if pbs.ProcessedBytes > 0 && currentSpeedBytesPerSec > 0 {
		remainingBytes := pbs.TotalBytes - pbs.ProcessedBytes
		etaSeconds := float64(remainingBytes) / currentSpeedBytesPerSec
		etaStr = fmt.Sprintf("%02d:%02d:%02d remaining",
			int(etaSeconds/3600),
			int(etaSeconds/60)%60,
			int(etaSeconds)%60)
	} else {
		etaStr = "calculating..."
	}

	// Update last values for next speed calculation
	pbs.LastUpdateTime = time.Now()
	pbs.LastProcessedBytes = pbs.ProcessedBytes

	// Clear the current line and print the new progress
	// \r moves the cursor to the beginning of the line
	// We print spaces to clear any leftover characters from a previous longer line
	fmt.Fprintf(os.Stdout, "\r[INFO] Progress: [%s] %3.0f%% (%s/%s) | Files Found: %d | @ %.2fMB/s [%s]    ",
		bar,
		percentage,
		format.FormatBytes(pbs.ProcessedBytes),
		format.FormatBytes(pbs.TotalBytes),
		pbs.FilesFound,
		currentSpeedMBps,
		etaStr)

	// Ensure the buffer is flushed to the terminal immediately
	os.Stdout.Sync()
}

// ClearLine prints a newline, effectively finishing the progress bar output
func (pbs *ProgressBarState) Finish() {
	fmt.Println() // Move to the next line after the bar is done
}
