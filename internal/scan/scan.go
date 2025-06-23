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
package scan

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ostafen/digler/internal/disk"
	"github.com/ostafen/digler/internal/env"
	"github.com/ostafen/digler/internal/format"
	"github.com/ostafen/digler/internal/fs"
	"github.com/ostafen/digler/pkg/dfxml"
	fmtutil "github.com/ostafen/digler/pkg/util/format"
)

type Options struct {
	DumpDir        string
	ReportFile     string
	MaxScanSize    uint64
	ScanBufferSize uint64
	BlockSize      uint64
	MaxFileSize    uint64
	DisableLog     bool
	FileExt        []string
	LogLevel       slog.Level
}

func Scan(filePath string, opts Options) error {
	partitions, err := DiscoverPartitions(filePath)
	if err != nil {
		return err
	}

	scanAllPartitions := true
	partitionsToScan := map[int]bool{}

	for _, p := range partitions {
		if scanAllPartitions || partitionsToScan[p.Num] {
			if err := ScanPartition(&p, filePath, opts); err != nil {
				return err
			}
		}
	}
	return nil
}

func absPath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
}

func ScanPartition(p *disk.Partition, filePath string, opts Options) error {
	f, err := fs.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	imgInfo, err := f.Stat()
	if err != nil {
		return err
	}

	session := GenSessionID()

	var reportFileName string
	if opts.ReportFile == "" {
		reportFileName = fmt.Sprintf("report_%s.xml", session)
	}

	outFile, err := os.Create(reportFileName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	reportFileWriter := dfxml.NewDFXMLWriter(outFile)
	defer reportFileWriter.Close()

	err = reportFileWriter.WriteHeader(dfxml.DFXMLHeader{
		XmlOutput: dfxml.XmlOutputVersion,
		Metadata:  dfxml.DefaultMetadata,
		Creator: dfxml.Creator{
			Package:              env.AppName,
			Version:              env.Version,
			ExecutionEnvironment: dfxml.GetExecEnv(),
		},
		Source: dfxml.Source{
			ImageFilename: filePath,
			SectorSize:    int(p.BlockSize),
			ImageSize:     uint64(imgInfo.Size()),
		},
	})
	if err != nil {
		return err
	}

	var logFilePath string
	if !opts.DisableLog {
		logFilePath = absPath(filepath.Join(opts.DumpDir, session) + ".log")
	}

	headers, err := format.FileHeaders(opts.FileExt...)
	if err != nil {
		return err
	}

	registry := format.BuildFileRegistry(headers...)

	fileExts := make([]string, len(headers))
	for i := range headers {
		fileExts[i] = headers[i].Ext
	}

	fmt.Println("[INFO] Starting scanning operation...")
	fmt.Printf("[INFO] Source: \t%s\n", absPath(filePath))
	fmt.Printf("[INFO] File Types: \t%s\n", strings.Join(fileExts, ","))

	if opts.DumpDir != "" {
		fmt.Printf("[INFO] Destination: \t%s\n", absPath(opts.DumpDir))
	}

	outLog := "disabled"
	if !opts.DisableLog {
		outLog = logFilePath
	}
	fmt.Printf("[INFO] Output Log: \t%s\n", outLog)
	fmt.Printf("[INFO] Scanning for %d signatures...\n", registry.Signatures())

	size := min(opts.MaxScanSize, p.Size)
	r := io.NewSectionReader(f, int64(p.Offset), int64(size))

	if opts.DumpDir != "" {
		if err := os.MkdirAll(opts.DumpDir, 0755); err != nil {
			return err
		}
	}

	logger, logFile, err := setupLogger(logFilePath, opts.LogLevel)
	if err != nil {
		return err
	}
	if logFile != nil {
		defer logFile.Close()
	}

	start := time.Now()
	filesFound := 0
	var totalDataSize uint64 = 0

	sc := format.NewScanner(
		logger,
		registry,
		int(opts.ScanBufferSize),
		int(p.BlockSize),
		opts.MaxFileSize,
	)
	for finfo := range sc.Scan(r, size) {
		filesFound++
		totalDataSize += finfo.Size

		if opts.DumpDir != "" {
			fileReader := io.NewSectionReader(r, int64(finfo.Offset), int64(finfo.Size))
			err := dumpFile(opts.DumpDir, finfo.Name, fileReader)
			if err != nil {
				return err
			}
		}

		err := reportFileWriter.WriteFileObject(dfxml.FileObject{
			Filename: finfo.Name,
			FileSize: uint64(finfo.Size),
			ByteRuns: dfxml.ByteRuns{
				Runs: []dfxml.ByteRun{{
					Offset:    uint64(finfo.Offset),
					ImgOffset: uint64(finfo.Offset),
					Length:    uint64(finfo.Size),
				}},
			},
		})
		if err != nil {
			logger.Error("unable to write index entry", "err", err)
		}
	}

	fmt.Println()

	fmt.Printf("[INFO] Scan completed!\n")
	fmt.Printf("[INFO] Files found: \t%d\n", filesFound)
	fmt.Printf("[INFO] Total data: \t%s\n", fmtutil.FormatBytes(int64(size)))
	fmt.Printf("[INFO] Duration: \t%s\n", FormatDurationHMS(time.Since(start)))
	fmt.Printf("[INFO] Report saved to: \t%s\n", absPath(reportFileName))

	if !opts.DisableLog {
		fmt.Printf("[INFO] Detailed scan log: \t%s\n", logFilePath)
	}
	return nil
}

func dumpFile(dumpDir string, fileName string, r io.Reader) error {
	f, err := os.Create(filepath.Join(dumpDir, fileName))
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", fileName, err)
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 1024*1024) // 1MB buffer

	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return w.Flush()
}

func DiscoverPartitions(path string) ([]disk.Partition, error) {
	imgFile, err := fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file %q: %w", path, err)
	}
	defer imgFile.Close()

	var firstSector [512]byte
	_, err = imgFile.ReadAt(firstSector[:], 0)
	if err != nil {
		return nil, err
	}

	// Try to parse the first sector as an MBR
	mbr, err := disk.ParseMBR(firstSector[:])
	if err == nil {
		mbrPartitions, err := GetMBRPartitions(imgFile, mbr)
		if err != nil {
			return nil, err
		}
		if len(mbrPartitions) > 0 {
			return mbrPartitions, nil
		}
	}

	// TODO: if was unable to determine the partitions,
	// Maybe, try to locate partition boundaries using FAT signature, etc?

	finfo, err := imgFile.Stat()
	if err != nil {
		return nil, err
	}

	return []disk.Partition{
		fullDiskPartition(uint64(finfo.Size())),
	}, nil
}

func fullDiskPartition(diskSize uint64) disk.Partition {
	return disk.Partition{
		FSType:    1,
		Num:       0,
		Offset:    0,
		Size:      diskSize,
		BlockSize: disk.DefaultBlocksize,
	}
}

func GetMBRPartitions(imgFile fs.File, mbr *disk.MBR) ([]disk.Partition, error) {
	// protective MBR for GPT disks
	if p := mbr.PartitionEntries[0]; p.PartitionType == disk.PartitionTypeGPT {
		offset := int64(p.ReadStartLBA()) * disk.DefaultBlocksize
		size := uint64(binary.LittleEndian.Uint32(p.TotalSectors[:])) * uint64(disk.DefaultBlocksize)

		// TODO: discover sector size
		return []disk.Partition{
			{
				FSType:    0,
				Num:       0,
				Offset:    uint64(offset),
				BlockSize: disk.DefaultBlocksize,
				Size:      size,
			},
		}, nil
	}

	partitions := make([]disk.Partition, 0, len(mbr.PartitionEntries))
	for n, p := range mbr.PartitionEntries {
		switch p.PartitionType {
		case disk.PartitionTypeFAT12,
			disk.PartitionTypeFAT16LessThan32MB,
			disk.PartitionTypeFAT16GreaterThan32MB,
			disk.PartitionTypeFAT16LBA,
			disk.PartitionTypeFAT32LBA,
			disk.PartitionTypeFAT32CHS:

			offset := int64(p.ReadStartLBA()) * disk.DefaultBlocksize

			var buf [512]byte
			_, err := imgFile.ReadAt(buf[:], offset)
			if err != nil {
				continue
			}

			fatSector, err := disk.ReadFatBootSectorFrom(buf[:])
			if err == nil {
				partitions = append(partitions, disk.Partition{
					FSType:    0,
					Num:       n,
					Offset:    uint64(offset),
					BlockSize: uint32(fatSector.SectorSize),
					Size:      uint64(binary.LittleEndian.Uint32(p.TotalSectors[:])) * uint64(fatSector.SectorSize),
				})
			}
		}
	}
	return partitions, nil
}

// GenSessionID creates a unique file name for a scan session.
// The format is "scan_YYYYMMDD_HHMMSS".
func GenSessionID() string {
	now := time.Now()

	// Format the time as YYYYMMDD_HHMMSS
	// YYYY = year (e.g., 2025)
	// MM   = month (e.g., 05)
	// DD   = day (e.g., 30)
	// HH   = hour (24-hour format, e.g., 16)
	// MM   = minute (e.g., 03)
	// SS   = second (e.g., 20)
	return now.Format("20060102_150405") // Go's special reference time format
}

// FormatDurationHMS formats a time.Duration into HH:MM:SS string.
// It handles durations that might be less than an hour or greater than 24 hours.
func FormatDurationHMS(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	totalSeconds := int64(d.Seconds())

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// setupLogger initializes a new slog.Logger that writes to a specified file or discards output.
// - logFilePath: The full path to the log file. If empty, logs will be discarded (file logging disabled).
// - minLevel: The minimum log level to write.
// It returns the logger instance and the *os.File, which will be nil if logging to file is disabled.
// The returned *os.File (if not nil) should be closed by the caller.
func setupLogger(logFilePath string, minLevel slog.Level) (*slog.Logger, *os.File, error) {
	var writer io.Writer
	var file *os.File

	if logFilePath == "" {
		writer = io.Discard
	} else {
		logDir := filepath.Dir(logFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create log directory %q: %w", logDir, err)
		}

		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file %q: %w", logFilePath, err)
		}
		writer = f
		file = f
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level:     minLevel,
		AddSource: true,
	})

	logger := slog.New(handler)
	return logger, file, nil
}
