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
package dfxml

import (
	"encoding/xml"
	"io"
)

// ReadFileObjects parses and returns all <fileobject> elements from the reader.
func ReadFileObjects(r io.Reader) ([]FileObject, error) {
	dec := xml.NewDecoder(r)
	var fileObjects []FileObject

	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// Look for start elements named "fileobject"
		if startElem, ok := tok.(xml.StartElement); ok && startElem.Name.Local == "fileobject" {
			var fo FileObject
			if err := dec.DecodeElement(&fo, &startElem); err != nil {
				return nil, err
			}
			fileObjects = append(fileObjects, fo)
		}
	}
	return fileObjects, nil
}
