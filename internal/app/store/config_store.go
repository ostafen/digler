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
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	osutils "github.com/ostafen/digler/pkg/util/os"
)

type ConfigStore struct {
	mu             sync.RWMutex
	configFilePath string

	config map[string]string
}

func NewConfigStore(path string) (*ConfigStore, error) {
	s := &ConfigStore{
		configFilePath: path,
		config:         make(map[string]string),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return nil, err
			}
			return s, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &s.config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return s, nil
}

func (s *ConfigStore) GetOrSet(param string, defaultValue string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v := s.config[param]
	if v == "" {
		err := s.set(param, defaultValue)
		return defaultValue, err
	}
	return v, nil
}

func (s *ConfigStore) Get(param string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v := s.config[param]
	return v, nil
}

func (s *ConfigStore) Set(param, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.set(param, value)
}

func (s *ConfigStore) set(param, value string) error {
	if value == "" {
		return fmt.Errorf("value cannot be empty")
	}

	s.config[param] = value
	return s.save()
}

func (s *ConfigStore) Delete(param string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, has := s.config[param]
	if !has {
		return nil
	}
	delete(s.config, param)

	return s.save()
}

func (s *ConfigStore) save() error {
	return osutils.AtomicWriteFile(s.configFilePath, func(f *os.File) error {
		enc := json.NewEncoder(f)
		enc.SetIndent("", " ")
		return enc.Encode(s.config)
	})
}

func (s *ConfigStore) Close() error {
	return nil
}
