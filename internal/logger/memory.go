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
package logger

import (
	"bytes"
	"sync"
)

// MemoryWriter is an in-memory writer that stores log lines
type MemoryWriter struct {
	mu    sync.Mutex
	lines []string
	buf   bytes.Buffer
}

// Write implements io.Writer
func (m *MemoryWriter) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	n, err = m.buf.Write(p)
	if err != nil {
		return n, err
	}

	// Split full lines from the buffer
	for {
		data := m.buf.Bytes()
		idx := bytes.IndexByte(data, '\n')
		if idx == -1 {
			break
		}

		line := string(data[:idx])
		m.lines = append(m.lines, line)
		m.buf.Next(idx + 1) // Remove processed line including '\n'
	}

	return n, nil
}

// Lines returns all written lines
func (m *MemoryWriter) Lines() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.lines...) // return a copy
}

// PopLines returns all written lines and removes them from the buffer
func (m *MemoryWriter) PopLines() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	lines := m.lines
	m.lines = nil
	return lines
}

// NewInMemory creates a logger that writes to an in-memory buffer
func NewInMemory(level Level) (*Logger, *MemoryWriter) {
	mw := &MemoryWriter{}
	logger := New(mw, level)
	return logger, mw
}
