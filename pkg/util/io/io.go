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
package io

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// CopyFile copies data from the provided reader to the file at filePath.
// It creates or truncates the file and writes using a 32KB buffer.
func CopyFile(filePath string, r io.Reader) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", filePath, err)
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 32*1024)
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return w.Flush()
}
