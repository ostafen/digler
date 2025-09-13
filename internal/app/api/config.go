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
	"github.com/ostafen/digler/internal/app/store"
)

type ConfigAPI struct {
	store *store.ConfigStore
}

func NewConfigAPI(store *store.ConfigStore) *ConfigAPI {
	return &ConfigAPI{store: store}
}

func (s *ConfigAPI) SetConfig(param, value string) error {
	return s.store.Set(param, value)
}

func (s *ConfigAPI) GetOrSet(param, defaultValue string) (string, error) {
	return s.store.GetOrSet(param, defaultValue)
}

func (s *ConfigAPI) GetConfig(param string) (string, error) {
	return s.store.Get(param)
}

func (s *ConfigAPI) DeleteConfig(param string) error {
	return s.store.Delete(param)
}
