package format

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
)

var ErrChunkOrderError = fmt.Errorf("invalid PNG chunk order")

const (
	dsStart = iota
	dsSeenIHDR
	dsSeenPLTE
	dsSeentRNS
	dsSeenIDAT
	dsSeenIEND
)

const pngHeader = "\x89PNG\r\n\x1a\n"

type decoder struct {
	r     io.Reader
	crc   hash.Hash32
	stage int
	tmp   [3 * 256]byte
}

func (d *decoder) checkHeader() error {
	_, err := io.ReadFull(d.r, d.tmp[:len(pngHeader)])
	if err != nil {
		return err
	}

	if string(d.tmp[:len(pngHeader)]) != pngHeader {
		return fmt.Errorf("not a PNG file")
	}
	return nil
}

func (d *decoder) parseChunk() error {
	// Read the length and chunk type.
	if _, err := io.ReadFull(d.r, d.tmp[:8]); err != nil {
		return err
	}
	length := binary.BigEndian.Uint32(d.tmp[:4])
	d.crc.Reset()
	d.crc.Write(d.tmp[4:8])

	writeCheck := func(write bool) error {
		if write {
			if _, err := io.ReadFull(d.r, d.tmp[:length]); err != nil {
				return err
			}
			d.crc.Write(d.tmp[:length])
		}
		return d.verifyChecksum()
	}

	// Read the chunk data.
	switch string(d.tmp[4:8]) {
	case "IHDR":
		if d.stage != dsStart {
			return ErrChunkOrderError
		}
		d.stage = dsSeenIHDR
		return writeCheck(true)
	case "PLTE":
		if d.stage != dsSeenIHDR {
			return ErrChunkOrderError
		}
		d.stage = dsSeenPLTE
		return writeCheck(true)
	case "tRNS":
		d.stage = dsSeentRNS
		return writeCheck(true)
	case "IDAT":
		if d.stage < dsSeenIHDR || d.stage > dsSeenIDAT { //|| (d.stage == dsSeenIHDR && cbPaletted(d.cb)) {
			return ErrChunkOrderError
		} else if d.stage == dsSeenIDAT {
			// Ignore trailing zero-length or garbage IDAT chunks.
			//
			// This does not affect valid PNG images that contain multiple IDAT
			// chunks, since the first call to parseIDAT below will consume all
			// consecutive IDAT chunks required for decoding the image.
			break
		}
		d.stage = dsSeenIDAT

		for n := 0; n < int(length); n += len(d.tmp) {
			m := min(len(d.tmp), int(length)-n)

			_, err := io.ReadFull(d.r, d.tmp[:m])
			if err != nil {
				return err
			}
			d.crc.Write(d.tmp[:m])
		}
		return writeCheck(false)
	case "IEND":
		if d.stage != dsSeenIDAT {
			return ErrChunkOrderError
		}
		d.stage = dsSeenIEND
		return writeCheck(true)
	}

	if length > 0x7fffffff {
		return fmt.Errorf("bad chunk length: %d", length)
	}
	// Ignore this chunk (of a known length).
	var ignored [4096]byte
	for length > 0 {
		n, err := io.ReadFull(d.r, ignored[:min(len(ignored), int(length))])
		if err != nil {
			return err
		}
		d.crc.Write(ignored[:n])
		length -= uint32(n)
	}
	return d.verifyChecksum()
}

func (d *decoder) verifyChecksum() error {
	if _, err := io.ReadFull(d.r, d.tmp[:4]); err != nil {
		return err
	}
	if binary.BigEndian.Uint32(d.tmp[:4]) != d.crc.Sum32() {
		return fmt.Errorf("invalid checksum")
	}
	return nil
}

func ScanPNG(r *Reader) (uint64, error) {
	d := &decoder{
		r:   r,
		crc: crc32.NewIEEE(),
	}

	if err := d.checkHeader(); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return 0, err
	}

	for d.stage != dsSeenIEND {
		if err := d.parseChunk(); err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return 0, err
		}
	}
	return r.BytesRead(), nil
}
