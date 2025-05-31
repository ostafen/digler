package format

type FileHeader struct {
	Ext        string // File extension, e.g., "mp3", "wav"
	Signatures [][]byte
	ScanFile   func(r *Reader) (uint64, error)
}

var fileHeaders = []FileHeader{
	// audio formats
	mp3FileHeader,
	wavFileHeader,
	sunAudioFileHeader,
	wmaFileHeader,
	// image formats
	jpegFileHeader,
	pngFileHeader,
	gifFileHeader,
	// generic/documents formats
	zipFileHeader,
	pdfFileHeader,
}

func BuildFileRegistry() *FileRegistry {
	r := NewFileRegisty()
	for _, hdr := range fileHeaders {
		r.Add(hdr)
	}
	return r
}

func (r *FileRegistry) Formats() []string {
	formats := make([]string, len(fileHeaders))
	for i := range formats {
		formats[i] = fileHeaders[i].Ext
	}
	return formats
}

func (r *FileRegistry) Signatures() int {
	n := 0
	for _, hdr := range fileHeaders {
		n += len(hdr.Signatures)
	}
	return n
}
