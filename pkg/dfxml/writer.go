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

// DFXMLWriter provides methods for writing DFXML elements to an io.Writer.
type DFXMLWriter struct {
	w   io.Writer    // The underlying writer (e.g., os.Stdout, a file).
	enc *xml.Encoder // The XML encoder used to write XML elements.
}

// NewDFXMLWriter creates and initializes a new DFXMLWriter.
// It sets up the XML encoder to indent output with two spaces for readability.
func NewDFXMLWriter(w io.Writer) *DFXMLWriter {
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ") // Indent with two spaces for pretty printing.

	return &DFXMLWriter{
		w:   w,
		enc: enc,
	}
}

// WriteHeader writes the DFXML header, including the XML declaration and the root <dfxml> tag.
func (w *DFXMLWriter) WriteHeader(hdr DFXMLHeader) error {
	// Write XML header (e.g., <?xml version="1.0" encoding="UTF-8"?>)
	_, _ = w.w.Write([]byte(xml.Header))

	// Manually construct and encode the starting <dfxml> tag to include attributes like xmloutputversion.
	start := xml.StartElement{
		Name: xml.Name{Local: "dfxml"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "xmloutputversion"}, Value: hdr.XmlOutput},
		},
	}

	// Encode the starting tag.
	if err := w.enc.EncodeToken(start); err != nil {
		return err
	}

	// Temporarily clear XmlOutput to prevent it from being marshaled again as an element
	// when encoding the rest of the header structure.
	out := hdr.XmlOutput
	hdr.XmlOutput = ""

	if err := w.enc.Encode(hdr); err != nil {
		return err
	}
	hdr.XmlOutput = out // Restore XmlOutput in the original struct.
	return nil
}

// WriteFileObject encodes and writes a FileObject struct as an XML element.
func (w *DFXMLWriter) WriteFileObject(obj FileObject) error {
	return w.enc.Encode(obj)
}

// Close closes the DFXML document by writing the closing </dfxml> tag and flushing the encoder.
func (w *DFXMLWriter) Close() error {
	// Write the closing </dfxml> tag.
	if err := w.enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "dfxml"}}); err != nil {
		return err
	}
	return w.enc.Flush()
}
