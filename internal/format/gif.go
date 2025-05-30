package format

import (
	"errors"
	"fmt"
	"io"
)

// Section indicators.
const (
	sExtension       = 0x21
	sImageDescriptor = 0x2C
	sTrailer         = 0x3B
)

// Extensions.
const (
	eText           = 0x01 // Plain Text
	eGraphicControl = 0xF9 // Graphic Control
	eComment        = 0xFE // Comment
	eApplication    = 0xFF // Application
)

// Masks
const (
	// Fields.
	fColorTable         = 1 << 7
	fInterlace          = 1 << 6
	fColorTableBitsMask = 7
)

type gifDecoder struct {
	loopCount int
	r         io.Reader

	width, height       int
	imageFields         byte
	hasGlobalColorTable bool // true if the global color table is present
	dataParsed          bool // true if the image descriptor has been parsed

	tmp [1024]byte // must be at least 768 so we can read color table
}

func ScanGIF(r *Reader) (uint64, error) {
	d := gifDecoder{
		loopCount: -1,
		r:         r,
	}

	err := d.readHeaderAndScreenDescriptor()
	if err != nil {
		return 0, err
	}

	for {
		c, err := readByte(d.r.(io.ByteReader))
		if err != nil {
			return 0, fmt.Errorf("gif: reading frames: %v", err)
		}
		switch c {
		case sExtension:
			if err = d.readExtension(); err != nil {
				return 0, err
			}
		case sImageDescriptor:
			if err = d.readImageDescriptor(); err != nil {
				return 0, err
			}

			//if len(d.image) == 1 {
			//	return nil
			//}

		case sTrailer:
			if !d.dataParsed {
				// If we haven't parsed the image descriptor, we can't have a valid image.
				return 0, errors.New("gif: missing image data")
			}
			return r.n, nil

		default:
			return 0, fmt.Errorf("gif: unknown block type: 0x%.2x", c)
		}
	}
}

func (d *gifDecoder) readExtension() error {
	extension, err := readByte(d.r.(io.ByteReader))
	if err != nil {
		return fmt.Errorf("gif: reading extension: %v", err)
	}
	size := 0
	switch extension {
	case eText:
		size = 13
	case eGraphicControl:
		return d.readGraphicControl()
	case eComment:
		// nothing to do but read the data.
	case eApplication:
		b, err := readByte(d.r.(io.ByteReader))
		if err != nil {
			return fmt.Errorf("gif: reading extension: %v", err)
		}
		// The spec requires size be 11, but Adobe sometimes uses 10.
		size = int(b)
	default:
		return fmt.Errorf("gif: unknown extension 0x%.2x", extension)
	}
	if size > 0 {
		if err := readFull(d.r, d.tmp[:size]); err != nil {
			return fmt.Errorf("gif: reading extension: %v", err)
		}
	}

	// Application Extension with "NETSCAPE2.0" as string and 1 in data means
	// this extension defines a loop count.
	if extension == eApplication && string(d.tmp[:size]) == "NETSCAPE2.0" {
		n, err := d.readBlock()
		if err != nil {
			return fmt.Errorf("gif: reading extension: %v", err)
		}
		if n == 0 {
			return nil
		}
		if n == 3 && d.tmp[0] == 1 {
			d.loopCount = int(d.tmp[1]) | int(d.tmp[2])<<8
		}
	}
	for {
		n, err := d.readBlock()
		if err != nil {
			return fmt.Errorf("gif: reading extension: %v", err)
		}
		if n == 0 {
			return nil
		}
	}
}

func (d *gifDecoder) readGraphicControl() error {
	if err := readFull(d.r, d.tmp[:6]); err != nil {
		return fmt.Errorf("gif: can't read graphic control: %s", err)
	}
	if d.tmp[0] != 4 {
		return fmt.Errorf("gif: invalid graphic control extension block size: %d", d.tmp[0])
	}
	if d.tmp[5] != 0 {
		return fmt.Errorf("gif: invalid graphic control extension block terminator: %d", d.tmp[5])
	}
	return nil
}

func (d *gifDecoder) parseImageDescriptorBounds() error {
	if err := readFull(d.r, d.tmp[:9]); err != nil {
		return fmt.Errorf("gif: can't read image descriptor: %s", err)
	}
	left := int(d.tmp[0]) + int(d.tmp[1])<<8
	top := int(d.tmp[2]) + int(d.tmp[3])<<8
	width := int(d.tmp[4]) + int(d.tmp[5])<<8
	height := int(d.tmp[6]) + int(d.tmp[7])<<8
	d.imageFields = d.tmp[8]

	// The GIF89a spec, Section 20 (Image Descriptor) says: "Each image must
	// fit within the boundaries of the Logical Screen, as defined in the
	// Logical Screen Descriptor."
	//
	// This is conceptually similar to testing
	//	frameBounds := image.Rect(left, top, left+width, top+height)
	//	imageBounds := image.Rect(0, 0, d.width, d.height)
	//	if !frameBounds.In(imageBounds) { etc }
	// but the semantics of the Go image.Rectangle type is that r.In(s) is true
	// whenever r is an empty rectangle, even if r.Min.X > s.Max.X. Here, we
	// want something stricter.
	//
	// Note that, by construction, left >= 0 && top >= 0, so we only have to
	// explicitly compare frameBounds.Max (left+width, top+height) against
	// imageBounds.Max (d.width, d.height) and not frameBounds.Min (left, top)
	// against imageBounds.Min (0, 0).
	if left+width > d.width || top+height > d.height {
		return errors.New("gif: frame bounds larger than image bounds")
	}
	return nil
}

func (d *gifDecoder) readImageDescriptor() error {
	if err := d.parseImageDescriptorBounds(); err != nil {
		return err
	}

	useLocalColorTable := d.imageFields&fColorTable != 0
	if useLocalColorTable {
		if err := d.skipColorTable(d.imageFields); err != nil {
			return err
		}
	} else if !d.hasGlobalColorTable {
		return errors.New("gif: no color table")
	}

	litWidth, err := readByte(d.r.(io.ByteReader))
	if err != nil {
		return fmt.Errorf("gif: reading image data: %v", err)
	}
	if litWidth < 2 || litWidth > 8 {
		return fmt.Errorf("gif: pixel size in decode out of range: %d", litWidth)
	}

	// discard LZW encoded blocks
	for {
		size, err := readByte(d.r.(io.ByteReader)) // read LZW minimum code size
		if err != nil {
			return fmt.Errorf("gif: reading image data: %v", err)
		}
		if size == 0 {
			// 0 means end of LZW data.
			break
		}
		if err := discard(int(size), d.r); err != nil {
			return err
		}
	}

	d.dataParsed = true
	return nil
}

func (d *gifDecoder) readBlock() (int, error) {
	n, err := readByte(d.r.(io.ByteReader))
	if n == 0 || err != nil {
		return 0, err
	}
	if err := readFull(d.r, d.tmp[:n]); err != nil {
		return 0, err
	}
	return int(n), nil
}

func (d *gifDecoder) readHeaderAndScreenDescriptor() error {
	err := readFull(d.r, d.tmp[:13])
	if err != nil {
		return fmt.Errorf("gif: reading header: %v", err)
	}
	version := string(d.tmp[:6])
	if version != "GIF87a" && version != "GIF89a" {
		return fmt.Errorf("gif: can't recognize format %q", version)
	}

	d.width = int(d.tmp[6]) + int(d.tmp[7])<<8
	d.height = int(d.tmp[8]) + int(d.tmp[9])<<8

	if fields := d.tmp[10]; fields&fColorTable != 0 {
		//d.backgroundIndex = d.tmp[11]
		// readColorTable overwrites the contents of d.tmp, but that's OK.
		d.hasGlobalColorTable = true

		if err := d.skipColorTable(fields); err != nil {
			return err
		}
	}
	// d.tmp[12] is the Pixel Aspect Ratio, which is ignored.
	return nil
}

func (d *gifDecoder) skipColorTable(fields byte) error {
	n := 1 << (1 + uint(fields&fColorTableBitsMask))
	err := readFull(d.r, d.tmp[:3*n])
	if err != nil {
		return fmt.Errorf("gif: reading color table: %s", err)
	}
	return nil
}

func readByte(r io.ByteReader) (byte, error) {
	b, err := r.ReadByte()
	if err != nil {
		if err == io.EOF {
			return 0, io.ErrUnexpectedEOF
		}
		return 0, fmt.Errorf("gif: reading byte: %v", err)
	}
	return b, nil
}

func readFull(r io.Reader, b []byte) error {
	_, err := io.ReadFull(r, b)
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return err
}
