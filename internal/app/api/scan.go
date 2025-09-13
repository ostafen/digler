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
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ostafen/digler/internal/app/store"
	"github.com/ostafen/digler/internal/format"
	"github.com/ostafen/digler/internal/fs"
	"github.com/ostafen/digler/internal/logger"
	"github.com/ostafen/digler/pkg/sysinfo"
	osutils "github.com/ostafen/digler/pkg/util/os"
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
	StartedAt  time.Time
	SourceType string
	Status     ScanStatus
	ScanID     string
	File       fs.File
	Scanner    *format.Scanner
	LogWriter  *logger.MemoryWriter
	FilesFound []FileInfo
	Recovery   *RecoverySession
}

type ScanInfo struct {
	ID              string `json:"id"`
	ScanStartedAt   uint64 `json:"scanStartedAt"`
	SourcePath      string `json:"sourcePath"`
	SourceType      string `json:"sourceType"`
	FilesFound      int    `json:"filesFound"`
	SignaturesFound int    `json:"signaturesFound"`
}

type ScanRecord struct {
	ScanInfo
	Files []FileInfo `json:"files"`
}

type ScanAPI struct {
	mu sync.RWMutex

	store          *store.HistoryStore[ScanRecord]
	currScanData   *ScanData
	scanInProgress bool
}

func NewScanAPI(store *store.HistoryStore[ScanRecord]) *ScanAPI {
	return &ScanAPI{
		store: store,
	}
}

func (s *ScanAPI) SetCurrentScan(scanID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if scanID == "" {
		return s.resetScan()
	}

	ts, err := strconv.ParseUint(scanID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid scan id: %s", scanID)
	}

	rec, err := s.store.Get(ts)
	if err != nil {
		return err
	}

	f, err := fs.Open(rec.SourcePath)
	if err != nil {
		return err
	}

	data := &ScanData{
		StartedAt:  time.Unix(int64(rec.ScanStartedAt), 0),
		Status:     ScanStatusDone,
		SourceType: rec.SourceType,
		ScanID:     scanID,
		File:       f,
		FilesFound: rec.Files,
	}
	s.currScanData = data
	return nil
}

func (s *ScanAPI) resetScan() error {
	data := s.currScanData
	if data == nil {
		return nil
	}

	if data.Status != ScanStatusScanning {
		return fmt.Errorf("cannot reset scan")
	}

	if data.File != nil {
		data.File.Close()
	}
	return nil
}

func (s *ScanAPI) StartScan(filePath string, outputDir string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.scanInProgress {
		return "", ErrScanInProgress
	}
	s.scanInProgress = true

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
		s.scanInProgress = false
		return "", err
	}

	size, isDevice, err := statFile(f, filePath)
	if err != nil {
		s.scanInProgress = false
		return "", err
	}

	var sourceType string = "image"
	if isDevice {
		sourceType = "device"
	}

	startedAt := time.Now().Truncate(time.Second)
	scanID := strconv.FormatInt(startedAt.Unix(), 10)

	ctx, cancel := context.WithCancel(context.Background())

	s.currScanData = &ScanData{
		Ctx:        ctx,
		CtxCancel:  cancel,
		StartedAt:  time.Now().Truncate(time.Second),
		SourceType: sourceType,
		Status:     ScanStatusScanning,
		ScanID:     scanID,
		File:       f,
		Scanner:    scanner,
		LogWriter:  logWriter,
	}

	filesFound := make([]FileInfo, 0, 10)

	go func() {
		defer func() {
			// TODO: file must be closed only when the scan is completely done
			cancel()

			s.mu.Lock()
			s.currScanData.Status = ScanStatusDone
			s.currScanData.FilesFound = filesFound
			s.scanInProgress = false
			s.mu.Unlock()

			err := s.store.Append(ScanRecord{
				ScanInfo: ScanInfo{
					ID:              scanID,
					ScanStartedAt:   uint64(startedAt.Unix()),
					SourcePath:      filePath,
					SourceType:      sourceType,
					FilesFound:      int(scanner.FilesFound()),
					SignaturesFound: int(scanner.SignaturesFound()),
				},
				Files: filesFound,
			}, uint64(startedAt.Unix()))
			if err != nil {
				log.Println("unable to store scan results")
			}
		}()

		for fi := range scanner.Scan(ctx, f, uint64(size)) {
			filesFound = append(filesFound, FileInfo(fi))
		}
	}()
	return scanID, nil
}

func (s *ScanAPI) PauseScan(scanID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.ensureScanStatus(scanID, ScanStatusScanning)
	if err != nil {
		return err
	}

	data.Scanner.Pause()

	s.currScanData.Status = ScanStatusPaused
	return nil
}

func (s *ScanAPI) ResumeScan(scanID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.ensureScanStatus(scanID, ScanStatusPaused)
	if err != nil {
		return err
	}
	data.Scanner.Resume()

	s.currScanData.Status = ScanStatusScanning
	return nil
}

func (s *ScanAPI) AbortScan(scanID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.ensureScanStatus(scanID, ScanStatusScanning, ScanStatusPaused)
	if err != nil {
		return err
	}

	if data.Status == ScanStatusPaused {
		data.Scanner.Resume()
	}
	data.CtxCancel()

	s.currScanData.Status = ScanStatusDone
	return nil
}

func (s *ScanAPI) ensureScanStatus(scanID string, statuses ...ScanStatus) (*ScanData, error) {
	data := s.currScanData
	if data == nil || data.ScanID != scanID {
		return nil, fmt.Errorf("no scan found with ID %s", scanID)
	}

	if !slices.Contains(statuses, data.Status) {
		return nil, fmt.Errorf("invalid scan status")
	}
	return data, nil
}

type ScanHistoryRecord struct {
	ScanInfo
	IsMissing bool `json:"isMissing"`
}

func (s *ScanAPI) LoadScanHistory(maxRecords int) ([]ScanHistoryRecord, error) {
	records, err := s.store.LoadLast(maxRecords)
	if err != nil {
		return nil, err
	}

	scanInfos := make([]ScanHistoryRecord, len(records))
	for i, rec := range records {
		scanInfos[i] = ScanHistoryRecord{
			ScanInfo: rec.ScanInfo,
			// TODO: this won't probably work on windows for devices.
			IsMissing: !osutils.FileExists(rec.SourcePath),
		}
	}
	return scanInfos, nil
}

func (s *ScanAPI) ClearScanHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.store.RemoveAll()
}

type ScanStatusResponse struct {
	Status     ScanStatus `json:"status"`
	Logs       []string   `json:"logs"`
	Progress   float64    `json:"progress"`
	Signatures uint64     `json:"signatures"`
	Files      uint64     `json:"files"`
}

func (s *ScanAPI) PollStatus(scanID string) (*ScanStatusResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := s.currScanData
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.ensureScanStatus(scanID, ScanStatusDone)
	if err != nil {
		return nil, err
	}

	return &ScanResultResponse{
		FilesFound: data.FilesFound,
	}, nil
}

func (s *ScanAPI) FileContent(scanID string, name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.ensureScanStatus(scanID, ScanStatusDone)
	if err != nil {
		return "", err
	}

	idx := slices.IndexFunc(data.FilesFound, func(fi FileInfo) bool {
		return fi.Name == name
	})
	if idx < 0 {
		return "", fmt.Errorf("file %s not found in scan %s", name, scanID)
	}

	fi := &data.FilesFound[idx]

	r := io.NewSectionReader(
		data.File,
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
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.ensureScanStatus(scanID, ScanStatusDone)
	if err != nil {
		return err
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
	s.currScanData.Recovery = task

	go func() {
		r := data.File
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

		s.mu.Lock()
		s.currScanData.Status = ScanStatusDone
		s.mu.Unlock()
	}()
	return nil
}

type RecoveryStatus struct {
	Progress  float64 `json:"progress"`
	Recovered uint64  `json:"recovered"`
	Errors    uint64  `json:"errors"`
}

func (s *ScanAPI) RecoveryProgress(scanID string) (*RecoveryStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.ensureScanStatus(scanID, ScanStatusRecovery, ScanStatusDone)
	if err != nil {
		return nil, err
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

func statFile(f fs.File, path string) (int64, bool, error) {
	devices, err := sysinfo.ListDevices()
	if err != nil {
		return -1, false, err
	}

	for _, dev := range devices {
		if path == dev.Path {
			return dev.Size, true, nil
		}
	}

	stat, err := f.Stat()
	if err != nil {
		return -1, false, err
	}
	return stat.Size(), false, nil
}
