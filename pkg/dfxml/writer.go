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
