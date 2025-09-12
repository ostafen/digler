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
package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/ostafen/digler/internal/format"
	"github.com/ostafen/digler/internal/fs"
	"github.com/ostafen/digler/internal/logger"
	"github.com/ostafen/digler/pkg/sysinfo"
)

var ErrScanInProgress = fmt.Errorf("a scan is already in progress")

const (
	DefaultBufferSize  = 4 * 1024 * 1024         // 4 MB
	DefaultBlockSize   = 512                     // 512 bytes
	DefaultMaxFileSize = 10 * 1024 * 1024 * 1024 // 10 GB
)

type ScanStatus string

const (
	ScanStatusScanning ScanStatus = "scanning"
	ScanStatusPaused   ScanStatus = "paused"
	ScanStatusRecovery ScanStatus = "recovery"
	ScanStatusError    ScanStatus = "error"
	ScanStatusDone     ScanStatus = "done"
)

type FileInfo struct {
	Name   string `json:"name"`
	Ext    string `json:"ext"`
	Offset uint64 `json:"offset"` // Offset in the file where the format starts
	Size   uint64 `json:"size"`   // Size of the format in bytes
}

type ScanData struct {
	Ctx        context.Context
	CtxCancel  context.CancelFunc
	Status     ScanStatus
	ScanID     string
	Reader     io.ReaderAt
	Scanner    *format.Scanner
	LogWriter  *logger.MemoryWriter
	FilesFound []FileInfo
	Recovery   *RecoverySession
}

func (s *ScanData) Copy() *ScanData {
	return &ScanData{
		Ctx:        s.Ctx,
		CtxCancel:  s.CtxCancel,
		Status:     s.Status,
		ScanID:     s.ScanID,
		Reader:     s.Reader,
		Scanner:    s.Scanner,
		LogWriter:  s.LogWriter,
		FilesFound: s.FilesFound,
		Recovery:   s.Recovery,
	}
}

type ScanAPI struct {
	currScanData   atomic.Pointer[ScanData]
	scanInProgress atomic.Bool
}

func (s *ScanAPI) StartScan(filePath string, outputDir string) (string, error) {
	scanID, err := s.runScan(filePath)
	if err != nil {
		return "", err
	}
	return scanID, nil
}

func (s *ScanAPI) PauseScan(scanID string) error {
	data, err := s.ensureScanStatus(scanID, ScanStatusScanning)
	if err != nil {
		return err
	}

	data.Scanner.Pause()

	newData := data.Copy()
	newData.Status = ScanStatusPaused
	if !s.currScanData.CompareAndSwap(data, newData) {
		return fmt.Errorf("unable to update status")
	}
	return nil
}

func (s *ScanAPI) ResumeScan(scanID string) error {
	data, err := s.ensureScanStatus(scanID, ScanStatusPaused)
	if err != nil {
		return err
	}
	data.Scanner.Resume()

	newData := data.Copy()
	newData.Status = ScanStatusScanning
	if !s.currScanData.CompareAndSwap(data, newData) {
		return fmt.Errorf("unable to update status")
	}
	return nil
}

func (s *ScanAPI) AbortScan(scanID string) error {
	data, err := s.ensureScanStatus(scanID, ScanStatusScanning, ScanStatusPaused)
	if err != nil {
		return err
	}

	if data.Status == ScanStatusPaused {
		data.Scanner.Resume()
	}
	data.CtxCancel()

	newData := data.Copy()
	newData.Status = ScanStatusDone
	if !s.currScanData.CompareAndSwap(data, newData) {
		return fmt.Errorf("unable to update status")
	}
	return nil
}

func (s *ScanAPI) ensureScanStatus(scanID string, statuses ...ScanStatus) (*ScanData, error) {
	data := s.currScanData.Load()
	if data == nil || data.ScanID != scanID {
		return nil, fmt.Errorf("no scan found with ID %s", scanID)
	}

	if !slices.Contains(statuses, data.Status) {
		return nil, fmt.Errorf("invalid scan status")
	}
	return data, nil
}

type ScanStatusResponse struct {
	Status     ScanStatus `json:"status"`
	Logs       []string   `json:"logs"`
	Progress   float64    `json:"progress"`
	Signatures uint64     `json:"signatures"`
	Files      uint64     `json:"files"`
}

func (s *ScanAPI) PollStatus(scanID string) (*ScanStatusResponse, error) {
	data := s.currScanData.Load()
	if data == nil || data.ScanID != scanID {
		return nil, fmt.Errorf("no logs found for scan ID %s", scanID)
	}

	lines := data.LogWriter.PopLines()

	return &ScanStatusResponse{
		Status:     data.Status,
		Logs:       lines,
		Progress:   data.Scanner.Progress(),
		Signatures: data.Scanner.SignaturesFound(),
		Files:      data.Scanner.FilesFound(),
	}, nil
}

type ScanResultResponse struct {
	FilesFound []FileInfo `json:"files"`
}

func (s *ScanAPI) ScanResult(scanID string) (*ScanResultResponse, error) {
	data, err := s.ensureScanStatus(scanID, ScanStatusDone)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &ScanResultResponse{
		FilesFound: data.FilesFound,
	}, nil
}

func (s *ScanAPI) FileContent(scanID string, name string) (string, error) {
	data := s.currScanData.Load()
	if data == nil || data.ScanID != scanID {
		return "", fmt.Errorf("no logs found for scan ID %s", scanID)
	}
	if data.Status != ScanStatusDone {
		return "", fmt.Errorf("scan %s is not completed yet", scanID)
	}

	idx := slices.IndexFunc(data.FilesFound, func(fi FileInfo) bool {
		return fi.Name == name
	})
	if idx < 0 {
		return "", fmt.Errorf("file %s not found in scan %s", name, scanID)
	}

	fi := &data.FilesFound[idx]

	r := io.NewSectionReader(
		data.Reader,
		int64(fi.Offset),
		int64(fi.Size),
	)

	content, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	encodedContent := base64.StdEncoding.EncodeToString(content)
	return encodedContent, nil
}

type RecoverySession struct {
	Errors     atomic.Uint64
	Recovered  atomic.Uint64
	BytesRead  atomic.Uint64
	TotalBytes uint64
}

func (s *ScanAPI) StartRecovery(scanID string, fileNames []string, outputDir string) error {
	data := s.currScanData.Load()
	if data == nil || data.ScanID != scanID {
		return fmt.Errorf("no logs found for scan ID %s", scanID)
	}
	if data.Status != ScanStatusDone {
		return fmt.Errorf("scan %s is not completed yet", scanID)
	}

	files := make(map[string]bool, len(fileNames))
	for _, name := range fileNames {
		files[name] = true
	}

	if len(files) == 0 {
		return fmt.Errorf("no files specified for recovery")
	}

	totalBytes := uint64(0)
	for _, fi := range data.FilesFound {
		if files[fi.Name] {
			totalBytes += fi.Size
		}
	}

	task := &RecoverySession{
		TotalBytes: totalBytes,
	}

	newData := &ScanData{
		Status:     ScanStatusRecovery,
		ScanID:     data.ScanID,
		Reader:     data.Reader,
		Scanner:    data.Scanner,
		LogWriter:  data.LogWriter,
		FilesFound: data.FilesFound,
		Recovery:   task,
	}
	if !s.currScanData.CompareAndSwap(data, newData) {
		return fmt.Errorf("another recovery task is already in progress")
	}

	go func() {
		r := data.Reader
		for _, fi := range data.FilesFound {
			if files[fi.Name] {
				err := recoverFile(r, &fi, outputDir)
				if err != nil {
					task.Errors.Add(1)
				} else {
					task.Recovered.Add(1)
				}
				task.BytesRead.Add(fi.Size)
			}
		}

		newData.Status = ScanStatusDone
		s.currScanData.Store(newData)
	}()
	return nil
}

type RecoveryStatus struct {
	Progress  float64 `json:"progress"`
	Recovered uint64  `json:"recovered"`
	Errors    uint64  `json:"errors"`
}

func (s *ScanAPI) RecoveryProgress(scanID string) (*RecoveryStatus, error) {
	data := s.currScanData.Load()
	if data == nil || data.ScanID != scanID {
		return nil, fmt.Errorf("no logs found for scan ID %s", scanID)
	}

	if data.Status != ScanStatusRecovery && data.Status != ScanStatusDone {
		return nil, fmt.Errorf("no recovery task in progress for scan %s", scanID)
	}

	if data.Recovery == nil {
		return nil, fmt.Errorf("no recovery session found for scan %s", scanID)
	}

	return &RecoveryStatus{
		Progress:  float64(data.Recovery.BytesRead.Load()) / float64(data.Recovery.TotalBytes),
		Recovered: data.Recovery.Recovered.Load(),
		Errors:    data.Recovery.Errors.Load(),
	}, nil
}

func recoverFile(r io.ReaderAt, fi *FileInfo, outDir string) error {
	sr := io.NewSectionReader(
		r,
		int64(fi.Offset),
		int64(fi.Size),
	)

	f, err := os.Create(filepath.Join(outDir, fi.Name))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, sr)
	return err
}

func (s *ScanAPI) runScan(filePath string) (string, error) {
	if !s.scanInProgress.CompareAndSwap(false, true) {
		return "", ErrScanInProgress
	}

	logger, logWriter := logger.NewInMemory(logger.InfoLevel)

	// TODO: make parameters configurable
	scanner := format.NewScanner(
		logger,
		format.DefaultRegistry,
		DefaultBufferSize,
		DefaultBlockSize,
		DefaultMaxFileSize,
	)

	f, err := fs.Open(filePath)
	if err != nil {
		s.scanInProgress.Store(false)
		return "", err
	}

	size, err := fileSize(f, filePath)
	if err != nil {
		s.scanInProgress.Store(false)
		return "", err
	}

	scanID := uuid.NewString()

	ctx, cancel := context.WithCancel(context.Background())

	s.currScanData.Store(&ScanData{
		Ctx:       ctx,
		CtxCancel: cancel,
		Status:    ScanStatusScanning,
		ScanID:    scanID,
		Reader:    f,
		Scanner:   scanner,
		LogWriter: logWriter,
	})

	filesFound := make([]FileInfo, 0, 10)

	go func() {
		defer func() {
			// TODO: file must be closed only when the scan is completely done
			cancel()

			s.currScanData.Store(&ScanData{
				Ctx:        ctx,
				CtxCancel:  cancel,
				Status:     ScanStatusDone,
				ScanID:     scanID,
				Reader:     f,
				Scanner:    scanner,
				LogWriter:  logWriter,
				FilesFound: filesFound,
			})
			s.scanInProgress.Store(false)
		}()

		for fi := range scanner.Scan(ctx, f, uint64(size)) {
			filesFound = append(filesFound, FileInfo(fi))
		}
	}()
	return scanID, nil
}

func fileSize(f fs.File, path string) (int64, error) {
	devices, err := sysinfo.ListDevices()
	if err != nil {
		return -1, err
	}

	for _, dev := range devices {
		if path == dev.Path {
			return dev.Size, nil
		}
	}

	stat, err := f.Stat()
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}
