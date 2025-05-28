package format

type Header struct {
	Ext       string // File extension, e.g., "mp3", "wav"
	Signature []byte
	ScanFile  func([]byte) (uint64, error)
}

var DefaultHeaders = []Header{
	{
		Ext:      "mp3",
		ScanFile: ScanMP3,
	},
	{
		Ext:      "wav",
		ScanFile: ScanWAV,
	},
	{
		Ext:      "au",
		ScanFile: ScanSunAudio,
	},
	{
		Ext:      "wma",
		ScanFile: ScanWMA,
	},
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
	},
}
