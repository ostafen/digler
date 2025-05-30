package format

type FileHeader struct {
	Ext        string // File extension, e.g., "mp3", "wav"
	Signatures [][]byte
	ScanFile   func(r *Reader) (uint64, error)
}

var SupportedFileHeaders = []FileHeader{
	{
		Ext: "mp3",
		Signatures: [][]byte{
			{0xFF, 0xFA},
			{0xFF, 0xFB},
			{0xFF, 0xF2},
			{0xFF, 0xF3},
			{0xFF, 0xE2},
			{0xFF, 0xE3},
			[]byte("ID3"),
		},
		ScanFile: ScanMP3,
	},
	{
		Ext: "wav",
		Signatures: [][]byte{
			[]byte("RIFF"),
			[]byte("RIFX"),
		},
		ScanFile: ScanWAV,
	},
	{
		Ext: "au",
		Signatures: [][]byte{
			{0x2E, 0x73, 0x6E, 0x64},
		},
		ScanFile: ScanSunAudio,
	},
	{
		Ext: "wma",
		Signatures: [][]byte{
			asfHeaderGUID,
		},
		ScanFile: ScanWMA,
	},
	{
		Ext: "jpeg",
		Signatures: [][]byte{
			{0xFF, 0xD8, 0xFF},
		},
		ScanFile: ScanJPEG,
	},
	{
		Ext:        "png",
		Signatures: [][]byte{[]byte(pngHeader)},
		ScanFile:   ScanPNG,
	},
	{
		Ext: "gif",
		Signatures: [][]byte{
			[]byte("GIF87a"),
			[]byte("GIF89a"),
		},
		ScanFile: ScanGIF,
	},
	{
		Ext: "zip",
		Signatures: [][]byte{
			{'P', 'K', 0x03, 0x04},
			{'P', 'K', '0', '0', 'P', 'K', 0x03, 0x04},
		},
		ScanFile: ScanZIP,
	},
	{
		Ext:        "pdf",
		Signatures: [][]byte{pdfHeader},
		ScanFile:   ScanPDF,
	},
}

func BuildFileRegistry() *FileRegistry {
	r := NewFileRegisty()
	for _, hdr := range SupportedFileHeaders {
		r.Add(hdr)
	}
	return r
}
