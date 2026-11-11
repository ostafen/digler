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

//go:build darwin
// +build darwin

package fs

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

// strictDevice mimics a macOS raw device: unaligned offset/length -> EINVAL,
// short read at the end of the device.
type strictDevice struct {
	data []byte
	ss   int64
}

func (d *strictDevice) ReadAt(p []byte, off int64) (int, error) {
	if off%d.ss != 0 || int64(len(p))%d.ss != 0 {
		return 0, fmt.Errorf("read: invalid argument")
	}
	if off >= int64(len(d.data)) {
		return 0, io.EOF
	}
	n := copy(p, d.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func TestAlignedReadAt(t *testing.T) {
	const ss = 512
	data := make([]byte, 10*ss)
	for i := range data {
		data[i] = byte(i * 7)
	}
	dev := &strictDevice{data: data, ss: ss}

	cases := []struct {
		name    string
		off     int64
		n       int
		wantErr error
	}{
		{"aligned", 1024, 1024, nil},
		{"unaligned length (the mp3 case)", 2 * ss, 1682, nil},
		{"unaligned offset", 100, 700, nil},
		{"spans sector boundary", ss - 3, 6, nil},
		{"tiny", 5, 1, nil},
		{"ends at device end", 9*ss + 10, ss - 10, nil},
		{"past device end", 9*ss + 10, ss, io.EOF},
		{"fully beyond device", 20 * ss, 10, io.EOF},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := make([]byte, c.n)
			n, err := alignedReadAt(dev, ss, p, c.off)
			if err != c.wantErr {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			end := c.off + int64(c.n)
			if end > int64(len(data)) {
				end = int64(len(data))
			}
			var want []byte
			if c.off < end {
				want = data[c.off:end]
			}
			if !bytes.Equal(p[:n], want) {
				t.Fatalf("got %d bytes, content mismatch (want %d)", n, len(want))
			}
		})
	}
}

func TestStrictDeviceRejectsUnaligned(t *testing.T) {
	// Control: the unwrapped device must fail on the mp3-shaped read.
	dev := &strictDevice{data: make([]byte, 4096), ss: 512}
	if _, err := dev.ReadAt(make([]byte, 1682), 1024); err == nil {
		t.Fatal("strict device accepted an unaligned read; test instrument is broken")
	}
}
