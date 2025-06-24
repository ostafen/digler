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
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ostafen/digler/internal/disk"
	"github.com/ostafen/digler/internal/env"
	"github.com/ostafen/digler/internal/format"
	"github.com/ostafen/digler/internal/fs"
	"github.com/ostafen/digler/internal/logger"
	"github.com/ostafen/digler/pkg/dfxml"
	fmtutil "github.com/ostafen/digler/pkg/util/format"
	ioutil "github.com/ostafen/digler/pkg/util/io"
)

type Options struct {
	DumpDir        string       // DumpDir is the directory where carved files will be dumped. If empty, files will not be dumped.
	ReportFile     string       // ReportFile is the path to the report file. If empty, a default name will be used.
	MaxScanSize    uint64       // MaxScanSize is the maximum number of bytes to scan. If 0, the entire partition will be scanned.
	ScanBufferSize uint64       // ScanBufferSize is the size of the buffer to use during scanning. If 0, a default size is used.
	BlockSize      uint64       // BlockSize is the size of a block to read from the disk. If 0, the default block size is used.
	MaxFileSize    uint64       // MaxFileSize is the maximum size of a carved file. If 0, no limit is applied.
	DisableLog     bool         // DisableLog disables logging to a file. If true, no log file will be created.
	FileExt        []string     // file extensions to parse, e.g. "jpg,png,txt"
	Plugins        []string     // paths to plugin .so files or directories containing plugins
	LogLevel       logger.Level // LogLevel specifies the minimum log level to write to the log file.
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

	scanID := GetScanID()

	var reportFileName string
	if opts.ReportFile == "" {
		reportFileName = fmt.Sprintf("report_%s.xml", scanID)
	}

	outFile, err := os.Create(reportFileName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	blockSize := p.BlockSize
	if opts.BlockSize != 0 {
		blockSize = uint32(opts.BlockSize)
	}

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
			SectorSize:    int(blockSize),
			ImageSize:     uint64(imgInfo.Size()),
		},
	})
	if err != nil {
		return err
	}

	var logFilePath string
	if !opts.DisableLog {
		logFilePath = absPath(filepath.Join(opts.DumpDir, scanID) + ".log")
	}

	scanners, err := format.GetFileScanners(opts.FileExt...)
	if err != nil {
		return err
	}

	var pluginScanners []format.FileScanner
	if len(opts.Plugins) > 0 {
		pluginScanners, err = format.LoadPlugins(opts.Plugins...)
		if err != nil {
			return err
		}
		scanners = append(scanners, pluginScanners...)
	}

	registry := format.BuildFileRegistry(scanners...)

	fileExts := make([]string, len(scanners))
	for i := range scanners {
		fileExts[i] = scanners[i].Ext()
	}

	logger, logFile, err := setupLogger(logFilePath, opts.LogLevel)
	if err != nil {
		return err
	}
	if logFile != nil {
		defer logFile.Close()
	}

	logger.Info("Starting scanning operation...")
	logger.Infof("Source: \t%s", absPath(filePath))
	logger.Infof("File Types: \t%s", strings.Join(fileExts, ","))

	if len(pluginScanners) > 0 {
		logger.Infof("Loaded %d plugins(s): \t%s", len(pluginScanners), strings.Join(opts.Plugins, ","))
	} else {
		logger.Infof("No plugin loaded")
	}

	if opts.DumpDir != "" {
		logger.Infof("Destination: \t%s", absPath(opts.DumpDir))
	}

	outLog := "disabled"
	if !opts.DisableLog {
		outLog = logFilePath
	}
	logger.Infof("Output Log: \t%s", outLog)
	logger.Infof("Scanning for %d signatures...", registry.Signatures())

	size := min(opts.MaxScanSize, p.Size)
	r := io.NewSectionReader(f, int64(p.Offset), int64(size))

	if opts.DumpDir != "" {
		if err := os.MkdirAll(opts.DumpDir, 0755); err != nil {
			return err
		}
	}

	start := time.Now()
	filesFound := 0
	var totalDataSize uint64 = 0

	sc := format.NewScanner(
		logger,
		registry,
		int(opts.ScanBufferSize),
		int(blockSize),
		opts.MaxFileSize,
	)
	for finfo := range sc.Scan(r, size) {
		filesFound++
		totalDataSize += finfo.Size

		if opts.DumpDir != "" {
			if err := DumpFile(r, opts.DumpDir, &finfo); err != nil {
				logger.Errorf("unable to dump file %s: %s", finfo.Name, err)
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
			logger.Errorf("unable to write index entry: %s", err)
		}
	}

	logger.Infof("Scan completed!")
	logger.Infof("Files found: \t%d", filesFound)
	logger.Infof("Total data: \t%s", fmtutil.FormatBytes(int64(size)))
	logger.Infof("Duration: \t%s", FormatDurationHMS(time.Since(start)))
	logger.Infof("Report saved to: \t%s", absPath(reportFileName))

	if !opts.DisableLog {
		logger.Infof("Detailed scan log: \t%s", logFilePath)
	}
	return nil
}

func DumpFile(r io.ReaderAt, outDir string, finfo *format.FileInfo) error {
	fileReader := io.NewSectionReader(r, int64(finfo.Offset), int64(finfo.Size))

	return ioutil.CopyFile(filepath.Join(outDir, finfo.Name), fileReader)
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

// GetScanID creates a unique file name for a scan session.
// The format is "scan_YYYYMMDD_HHMMSS".
func GetScanID() string {
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
func setupLogger(logFilePath string, minLevel logger.Level) (*logger.Logger, *os.File, error) {
	var w io.Writer = os.Stdout
	var file *os.File

	if logFilePath != "" {
		logDir := filepath.Dir(logFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create log directory %q: %w", logDir, err)
		}

		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file %q: %w", logFilePath, err)
		}

		w = io.MultiWriter(os.Stdout, f)
		file = f
	}

	logger := logger.New(w, logger.Level(minLevel))
	return logger, file, nil
}
