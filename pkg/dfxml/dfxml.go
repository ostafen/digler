package dfxml

import (
	"encoding/xml"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"time"

	"github.com/ostafen/digler/pkg/sysinfo"
)

const XmlOutputVersion = "1.0"

var DefaultMetadata = Metadata{
	Xmlns:    "http://www.forensicswiki.org/wiki/Category:Digital_Forensics_XML",
	XmlnsXsi: "http://www.w3.org/2001/XMLSchema-instance",
	XmlnsDC:  "http://purl.org/dc/elements/1.1/",
	Type:     "Carve Report",
}

// DFXMLHeader represents the root element of a DFXML document.
type DFXMLHeader struct {
	XMLName   xml.Name `xml:"dfxml"`                           // Specifies the XML element name as "dfxml".
	XmlOutput string   `xml:"xmloutputversion,attr,omitempty"` // The version of the DFXML XML schema, an attribute. "omitempty" means it will be omitted if empty.
	Metadata  Metadata `xml:"metadata"`                        // Contains metadata about the DFXML document.
	Creator   Creator  `xml:"creator"`                         // Describes the software that created the DFXML.
	Source    Source   `xml:"source"`                          // Describes the source of the forensic image.
}

// Metadata contains various metadata attributes for the DFXML document.
type Metadata struct {
	Xmlns    string `xml:"xmlns,attr"`     // XML Namespace for the DFXML schema.
	XmlnsXsi string `xml:"xmlns:xsi,attr"` // XML Namespace for XML Schema Instance.
	XmlnsDC  string `xml:"xmlns:dc,attr"`  // XML Namespace for Dublin Core.
	Type     string `xml:"dc:type"`        // The type of the DFXML document, e.g., "forensic_disk_image".
}

// Creator describes the software and environment used to generate the DFXML.
type Creator struct {
	Package              string  `xml:"package"`               // The name of the software package.
	Version              string  `xml:"version"`               // The version of the software package.
	ExecutionEnvironment ExecEnv `xml:"execution_environment"` // Details about the execution environment.
}

// ExecEnv provides information about the operating system and host where the DFXML was created.
type ExecEnv struct {
	OS      string `xml:"os_sysname"` // Operating system name (e.g., "Linux", "Windows").
	Release string `xml:"os_release"` // Operating system release version.
	Version string `xml:"os_version"` // Operating system kernel version.
	Host    string `xml:"host"`       // Hostname of the machine.
	Arch    string `xml:"arch"`       // Architecture of the machine (e.g., "x86_64").
	UID     int    `xml:"uid"`        // User ID under which the process ran.
	Start   string `xml:"start_time"` // Start time of the DFXML generation.
}

// Source describes the original forensic image or data source.
type Source struct {
	ImageFilename string `xml:"image_filename"` // The filename of the forensic image.
	SectorSize    int    `xml:"sectorsize"`     // The size of a sector in bytes.
	ImageSize     uint64 `xml:"image_size"`     // The total size of the image in bytes.
}

// --- FileObject Struct ---

// FileObject represents a single file or directory within the forensic image.
type FileObject struct {
	XMLName  xml.Name `xml:"fileobject"` // Specifies the XML element name as "fileobject".
	Filename string   `xml:"filename"`   // The name of the file.
	FileSize uint64   `xml:"filesize"`   // The size of the file in bytes.
	ByteRuns ByteRuns `xml:"byte_runs"`  // Contains information about the physical location of file data.
}

// ByteRuns is a collection of ByteRun entries.
type ByteRuns struct {
	Runs []ByteRun `xml:"byte_run"` // A slice of ByteRun structs, representing data extents.
}

// ByteRun describes a contiguous block of data within the image.
type ByteRun struct {
	Offset    uint64 `xml:"offset,attr"`     // Logical offset within the file object.
	ImgOffset uint64 `xml:"img_offset,attr"` // Physical offset within the disk image.
	Length    uint64 `xml:"len,attr"`        // Length of the byte run.
}

// GetExecEnv retrieves runtime information to populate the ExecEnv struct.
func GetExecEnv() ExecEnv {
	// Get OS information

	sinfo, err := sysinfo.Stat()
	if err != nil {
		sinfo = &sysinfo.SysUnknown
	}
	// On Linux, you might read /etc/os-release for more detailed info
	// On Windows, you might use syscalls or 'ver' command
	// For simplicity, we'll leave these as empty or provide a basic placeholder.
	// For more robust OS release/version, consider specialized libraries or platform-specific calls.

	// Get hostname
	host, err := os.Hostname()
	if err != nil {
		host = "unknown_host" // Fallback if hostname can't be determined
	}

	// Get architecture
	arch := runtime.GOARCH // e.g., "amd64", "arm64"

	// Get UID (User ID)
	uid := 0
	currentUser, err := user.Current()
	if err == nil {
		if uidInt, parseErr := strconv.Atoi(currentUser.Uid); parseErr == nil {
			uid = uidInt
		}
	}

	// Get start time in a format suitable for DFXML (ISO 8601 extended format with UTC)
	// DFXML often expects UTC time.
	startTime := time.Now().UTC().Format("2006-01-02T15:04:05Z") // YYYY-MM-DDTHH:MM:SSZ

	return ExecEnv{
		OS:      sinfo.Name,
		Release: sinfo.Release,
		Version: sinfo.Version,
		Host:    host,
		Arch:    arch,
		UID:     uid,
		Start:   startTime,
	}
}
