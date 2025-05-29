package format

type FileHeader struct {
	Ext        string // File extension, e.g., "mp3", "wav"
	Signatures [][]byte
	ScanFile   func(r *Reader) (uint64, error)
}

var DefaultHeaders = []FileHeader{
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
		Ext:        "wma",
		Signatures: [][]byte{asfHeaderGUID},
		ScanFile:   ScanWMA,
	},
	/*
		{
			Ext:      "jpeg",
			ScanFile: ScanJPEG,
		},
		{
			Ext:      "png",
			ScanFile: ScanPNG,
		},
		{
			Ext:      "gif",
			ScanFile: ScanGIF,
		},
		{
			Ext:      "zip",
			ScanFile: ScanZIP,
		},*/
}

func BuildRegistry() *FileRegistry {
	r := NewFileRegisty()
	for _, hdr := range DefaultHeaders {
		r.Add(hdr)
	}
	return r
}
