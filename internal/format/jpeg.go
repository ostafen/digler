package format

import (
	"fmt"
	"io"
)

var jpegFileHeader = FileHeader{
	Ext: "jpeg",
	Signatures: [][]byte{
		{0xFF, 0xD8, 0xFF},
	},
	ScanFile: ScanJPEG,
}

const (
	sof0Marker = 0xc0 // Start Of Frame (Baseline Sequential).
	sof1Marker = 0xc1 // Start Of Frame (Extended Sequential).
	sof2Marker = 0xc2 // Start Of Frame (Progressive).
	dhtMarker  = 0xc4 // Define Huffman Table.
	rst0Marker = 0xd0 // ReSTart (0).
	rst7Marker = 0xd7 // ReSTart (7).
	soiMarker  = 0xd8 // Start Of Image.
	eoiMarker  = 0xd9 // End Of Image.
	sosMarker  = 0xda // Start Of Scan.
	dqtMarker  = 0xdb // Define Quantization Table.
	driMarker  = 0xdd // Define Restart Interval.
	comMarker  = 0xfe // COMment.
	// "APPlication specific" markers aren't part of the JPEG spec per se,
	// but in practice, their use is described at
	// https://www.sno.phy.queensu.ca/~phil/exiftool/TagNames/JPEG.html
	app0Marker  = 0xe0
	app14Marker = 0xee
	app15Marker = 0xef
)

func discard(n int, r io.Reader) error {
	_, err := io.CopyN(io.Discard, r, int64(n))
	return err
}

// ScanJPEG attempts to validate a JPEG file from the beginning of the 'data'
// buffer and determine its total size. This function is adapted from the
// standard library's 'image/jpeg' package's internal scanning logic,
// specifically from the 'decode' function, but modified to exclusively
// focus on finding the End Of Image (EOI) marker to determine the file's
// boundaries for file carving purposes.
//
// It deviates from 'image/jpeg.DecodeConfig' as that function is designed
// to read only enough of the header to extract configuration (like dimensions)
// and does not typically scan to the EOI marker. This 'ScanJPEG' implementation
// navigates JPEG segments, handles byte stuffing (0xFF00), extraneous data
// (0xFF fill bytes), and restart markers (RSTn) in a robust way, mimicking
// libjpeg's leniency, to reliably locate the file's end.
//
// It returns the total size of the JPEG file (the offset of the EOI marker
// plus its 2-byte length) or the buffer's length if the file appears truncated.
// It returns an error if the file is malformed or doesn't start with an SOI marker.
func ScanJPEG(r *Reader) (uint64, error) {
	// Check for the Start Of Image marker.
	var tmp [2]byte

	_, err := r.Read(tmp[:])
	if err != nil {
		return 0, err
	}

	if tmp[0] != 0xff || tmp[1] != soiMarker {
		return 0, fmt.Errorf("missing SOI marker")
	}

	// Process the remaining segments until the End Of Image marker.
	for {
		_, err := r.Read(tmp[:])
		if err != nil {
			return 0, err
		}
		for tmp[0] != 0xff {
			// Strictly speaking, this is a format error. However, libjpeg is
			// liberal in what it accepts. As of version 9, next_marker in
			// jdmarker.c treats this as a warning (JWRN_EXTRANEOUS_DATA) and
			// continues to decode the stream. Even before next_marker sees
			// extraneous data, jpeg_fill_bit_buffer in jdhuff.c reads as many
			// bytes as it can, possibly past the end of a scan's data. It
			// effectively puts back any markers that it overscanned (e.g. an
			// "\xff\xd9" EOI marker), but it does not put back non-marker data,
			// and thus it can silently ignore a small number of extraneous
			// non-marker bytes before next_marker has a chance to see them (and
			// print a warning).
			//
			// We are therefore also liberal in what we accept. Extraneous data
			// is silently ignored.
			//
			// This is similar to, but not exactly the same as, the restart
			// mechanism within a scan (the RST[0-7] markers).
			//
			// Note that extraneous 0xff bytes in e.g. SOS data are escaped as
			// "\xff\x00", and so are detected a little further down below.
			tmp[0] = tmp[1]
			tmp[1], err = r.ReadByte()
			if err != nil {
				return 0, err
			}
		}
		marker := tmp[1]
		if marker == 0 {
			// Treat "\xff\x00" as extraneous data.
			continue
		}
		for marker == 0xff {
			// Section B.1.1.2 says, "Any marker may optionally be preceded by any
			// number of fill bytes, which are bytes assigned code X'FF'".
			marker, err = r.ReadByte()
			if err != nil {
				return 0, err
			}
		}
		if marker == eoiMarker { // End Of Image.
			return uint64(r.BytesRead()), nil
		}
		if rst0Marker <= marker && marker <= rst7Marker {
			// Figures B.2 and B.16 of the specification suggest that restart markers should
			// only occur between Entropy Coded Segments and not after the final ECS.
			// However, some encoders may generate incorrect JPEGs with a final restart
			// marker. That restart marker will be seen here instead of inside the processSOS
			// method, and is ignored as a harmless error. Restart markers have no extra data,
			// so we check for this before we read the 16-bit length of the segment.
			continue
		}

		// Read the 16-bit length of the segment. The value includes the 2 bytes for the
		// length itself, so we subtract 2 to get the number of remaining bytes.
		if _, err = r.Read(tmp[:]); err != nil {
			return 0, err
		}
		n := int(tmp[0])<<8 + int(tmp[1]) - 2
		if n < 0 {
			return 0, fmt.Errorf("short segment length")
		}

		switch marker {
		case sof0Marker, sof1Marker, sof2Marker,
			dhtMarker, dqtMarker, sosMarker,
			driMarker, app0Marker, app14Marker:
			err = discard(n, r)
		default:
			if app0Marker <= marker && marker <= app15Marker || marker == comMarker {
				err = discard(n, r)
			} else if marker < 0xc0 { // See Table B.1 "Marker code assignments".
				err = fmt.Errorf("unknown marker")
			} else {
				err = fmt.Errorf("unknown marker")
			}
		}
		if err != nil {
			return 0, err
		}
	}
}
