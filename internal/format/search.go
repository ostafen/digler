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
package format

import (
	"bytes"
	"io"
)

// SeekAt efficiently searches for a byte signature (`sig`) within the Reader's stream,
// up to a maximum of `n` bytes from the current reader position.
// It uses a circular buffer to handle cases where the signature might span across
// internal read buffer boundaries. The function attempts to position the reader
// right at the beginning of the found signature.
//
// Parameters:
//
//	r: The custom Reader interface to read from.
//	sig: The byte slice representing the signature to search for.
//	n: The maximum number of bytes to search from the current reader position.
//
// Returns:
//
//	bool: True if the signature is found, false otherwise.
//	error: An error if an I/O error occurs during reading, other than io.EOF.
func SeekAt(r *Reader, sig []byte, n int) (bool, error) {
	sigLen := len(sig)

	// `pad` ensures that enough bytes are kept from the end of the previous buffer
	// to potentially form the beginning of the signature with bytes from the next read.
	// This handles cases where the signature is split across read boundaries.
	pad := sigLen - 1
	buf := make([]byte, pad+r.BufferSize())

	offset := 0
	for offset < n {
		// If `offset` is greater than 0, it means we've read in previous iterations.
		// We copy the last `pad` bytes of the `buf` to the beginning. This is crucial
		// for handling signatures that might be split across `peek` operations, ensuring
		// the end of the previous peeked data can combine with the start of the next.
		if offset > 0 {
			copy(buf, buf[len(buf)-pad:])
		}

		peekBuf, err := r.Peek(len(buf) - pad)
		if err != nil && err != io.EOF {
			return false, err
		}

		m := len(peekBuf)
		copy(buf[pad:], peekBuf)

		if m > 0 {
			// Determine the portion of `buf` to search.
			// If `offset` is greater than 0, we're searching in a buffer that includes
			// the overlap from the previous read, so `searchBuf` starts from the beginning of `buf`.
			// Otherwise, we search only in the newly peeked portion (offset by `pad`).
			var searchBuf []byte
			if offset > 0 {
				searchBuf = buf[:pad+m]
			} else {
				searchBuf = buf[pad : pad+m]
			}

			if idx := bytes.Index(searchBuf, sig); idx >= 0 {
				// If the signature is found, calculate how many bytes to discard
				// to position the reader right at the beginning of the found signature.
				discard := idx
				if offset > 0 {
					// If there was an overlap, the `idx` is relative to `buf[0]`,
					// so we subtract `pad` to get the actual discard amount relative
					// to the reader's current position before this peek.
					discard -= pad
				}

				_, err = r.Discard(discard)
				return true, err
			}
		}

		if err == io.EOF {
			break
		}

		offset += m

		_, err = r.Discard(m)
		if err != nil {
			return false, err
		}
	}
	return false, nil
}
