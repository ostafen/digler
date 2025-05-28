package disk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"syscall" // For open flags like O_EXCL, and for ioctl on Linux
	"unsafe"  // Required for unsafe.Pointer in syscalls for ioctl
)

// --- Constants and Structs ---

// Define equivalent constants for testdisk_mode flags
const (
	// TESTDISK_O_RDWR indicates a preference for read-write access.
	// In the original C, this is a bit flag passed to the function.
	TESTDISK_O_RDWR = 1 << iota
	// TESTDISK_O_DIRECT indicates a preference for O_DIRECT, bypassing OS cache.
	// Its effectiveness and availability are OS/filesystem dependent.
	TESTDISK_O_DIRECT
	// Add other flags as needed from PhotoRec's testdisk_mode
)

// DefaultSectorSize is the assumed sector size for regular files or when
// a device's sector size cannot be determined.
const DefaultSectorSize = 512

// DiskInfo represents the opened disk device or image file,
// similar to PhotoRec's `disk_t` (simplified for Go).
type DiskInfo struct {
	DevicePath string   // The path to the device or file (e.g., "/dev/sda", "image.dd")
	SectorSize int64    // The physical or logical sector size in bytes
	RealSize   int64    // The total size of the disk/image in bytes
	AccessMode int      // The actual mode the file was opened with (e.g., os.O_RDONLY, os.O_RDWR)
	IsDevice   bool     // True if the path refers to a block device, false if a regular file
	file       *os.File // The underlying *os.File handle for operations
	Offset     uint64   // Offset to the start of actual data (for images, e.g., DOSEMU)
	// The original C `disk_t` has more fields like geometry, model, and function pointers
	// (pread, pwrite, etc.). In Go, these are handled by methods on DiskInfo or
	// by the *os.File methods directly (e.g., ReadAt, WriteAt).
}

// Close closes the underlying file handle associated with the DiskInfo.
func (d *DiskInfo) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}

// ReadAt reads data from the disk source at a specific offset.
func (d *DiskInfo) ReadAt(p []byte, off int64) (n int, err error) {
	if d.file == nil {
		return 0, fmt.Errorf("diskinfo: file handle is nil")
	}
	return d.file.ReadAt(p, off)
}

// WriteAt writes data to the disk source at a specific offset.
func (d *DiskInfo) WriteAt(p []byte, off int64) (n int, err error) {
	if d.file == nil {
		return 0, fmt.Errorf("diskinfo: file handle is nil")
	}
	// Ensure the device was opened in read-write mode for writing.
	if (d.AccessMode & os.O_RDWR) == 0 {
		return 0, fmt.Errorf("diskinfo: device not opened in read-write mode")
	}
	return d.file.WriteAt(p, off)
}

// --- OS-Specific Helpers (Linux Example) ---

// GetSectorSizeLinux attempts to retrieve the logical block/sector size
// of a Linux block device using the BLKSSZGET ioctl.
// This is Linux-specific.
func GetSectorSizeLinux(file *os.File) (int64, error) {
	var sectorSize uint32
	// syscall.S_BLKSIZE is a Linux ioctl command to get the logical block size.
	// For physical sector size, other ioctls or sysfs parsing might be needed.
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), syscall.S_BLKSIZE, uintptr(unsafe.Pointer(&sectorSize)))
	if errno != 0 {
		return 0, fmt.Errorf("ioctl BLKSSZGET failed: %w", errno)
	}
	return int64(sectorSize), nil
}

// GetDiskSizeLinux attempts to retrieve the total size in bytes
// of a Linux block device using the BLKGETSIZE64 ioctl.
// This is Linux-specific.
func GetDiskSizeLinux(file *os.File) (int64, error) {
	var size int64
	// BLKGETSIZE64 is a Linux ioctl command for getting device size in 64-bit bytes.
	const BLKGETSIZE64 = 0x80081272 // This constant is often used in Linux kernel headers.
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), BLKGETSIZE64, uintptr(unsafe.Pointer(&size)))
	if errno != 0 {
		return 0, fmt.Errorf("ioctl BLKGETSIZE64 failed: %w", errno)
	}
	return size, nil
}

// --- Image Format Magic Bytes (Simplified) ---

// evfFileSignature is the magic byte sequence for EnCase/Expert Witness Format files.
var evfFileSignature = [8]byte{'E', 'V', 'F', 0x09, 0x0D, 0x0A, 0xFF, 0x00}

// DosemuImageMagic is a placeholder for the DOSEMU image magic number.
// The actual value would be defined in PhotoRec's C headers.
const DosemuImageMagic uint32 = 0xFEEDABAA // Example magic value

// ReadFirstBlock reads the initial block of data from the file/device.
func ReadFirstBlock(file *os.File, blockSize int64) ([]byte, error) {
	buf := make([]byte, blockSize)
	n, err := file.ReadAt(buf, 0) // Read from offset 0
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read first block: %w", err)
	}
	return buf[:n], nil
}

// Stat attempts to open a disk device or image file,
// similar to PhotoRec's file_test_availability.
//
// devicePath:    The path to the disk device (e.g., "/dev/sda") or image file ("disk.img").
// verbose:       Controls the level of logging output (e.g., 0 for silent, >0 for info).
// testdiskMode:  Bit flags indicating desired access modes (e.g., TESTDISK_O_RDWR).
//
// Returns a *DiskInfo struct on success, or an error if the device cannot be opened
// or its properties cannot be determined.
func Stat(devicePath string, verbose int, testdiskMode int) (*DiskInfo, error) {
	diskInfo := &DiskInfo{
		DevicePath: devicePath,
		SectorSize: DefaultSectorSize, // Initial default, may be updated for devices
	}
	var file *os.File
	var err error
	var finalOpenFlags int = 0 // The actual flags used to open the file

	// --- 1. Attempt Read-Write Access (if TESTDISK_O_RDWR is set) ---
	if (testdiskMode & TESTDISK_O_RDWR) == TESTDISK_O_RDWR {
		// Attempt to open with O_RDWR and O_EXCL (exclusive access) first.
		// syscall.O_EXCL ensures that if the file/device already exists and is open,
		// this open call will fail.
		flagsToTry := os.O_RDWR | syscall.O_EXCL
		file, err = os.OpenFile(devicePath, flagsToTry, 0600) // 0600 is file permissions (rw- --- ---)

		if err != nil {
			// If O_EXCL fails due to device busy or invalid argument, retry without O_EXCL.
			// This mimics the C code's `errno==EBUSY || errno==EINVAL` check.
			if os.IsPermission(err) || strings.Contains(err.Error(), "resource busy") || strings.Contains(err.Error(), "invalid argument") {
				if verbose > 1 {
					fmt.Printf("Attempting R/W with O_EXCL failed for %s, retrying without O_EXCL: %v\n", devicePath, err)
				}
				flagsToTry = os.O_RDWR
				file, err = os.OpenFile(devicePath, flagsToTry, 0600)
			}
		}

		if err == nil { // If we successfully opened R/W
			finalOpenFlags = flagsToTry

			// On Linux, check if the device is actually read-only even if opened R/W.
			// This is for cases like read-only loop devices.
			if runtime.GOOS == "linux" {

				/*
					var readonly int32 // ioctl expects a pointer to an int32
					_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), syscall.BLKROGET, uintptr(unsafe.Pointer(&readonly)))
					if errno == 0 && readonly > 0 { // If ioctl succeeded and device is read-only
						if verbose > 0 {
							fmt.Printf("Device %s opened R/W but is physically read-only; falling back to read-only.\n", devicePath)
						}
						file.Close() // Close the read-write handle
						file = nil   // Mark as closed to trigger read-only attempt
						// Clear the RDWR flag to ensure subsequent attempts are read-only
						testdiskMode &^= TESTDISK_O_RDWR
					}*/
			}
		} else { // R/W opening failed
			// Check if failure indicates device truly unavailable (ENXIO, ENOENT, ENOMEDIUM)
			// This would prevent further read-only attempts.
			if os.IsNotExist(err) || strings.Contains(err.Error(), "no such device") || strings.Contains(err.Error(), "no medium") {
				// C code's equivalent of ENXIO, ENOENT, ENOMEDIUM
				// Here, we'll just let it proceed to try read-only or fail later if no other option
			}
		}
	}

	// --- 2. Fallback to Read-Only Access (if needed) ---
	if file == nil { // If no file handle from R/W attempt, or it was closed
		// The C code also tries O_EXCL then without, for O_RDONLY
		flagsToTry := os.O_RDONLY | syscall.O_EXCL
		file, err = os.OpenFile(devicePath, flagsToTry, 0600)

		if err != nil {
			if os.IsPermission(err) || strings.Contains(err.Error(), "resource busy") || strings.Contains(err.Error(), "invalid argument") {
				if verbose > 1 {
					fmt.Printf("Attempting R/O with O_EXCL failed for %s, retrying without O_EXCL: %v\n", devicePath, err)
				}
				flagsToTry = os.O_RDONLY
				file, err = os.OpenFile(devicePath, flagsToTry, 0600)
			}
		}
		if err == nil {
			finalOpenFlags = flagsToTry
		}
	}

	// --- 3. Final Check for Open Success and Error Logging ---
	if err != nil || file == nil {
		if verbose > 0 {
			fmt.Fprintf(os.Stderr, "file_test_availability %s: %v\n", devicePath, err)
		}
		// Special handling from C code for non-/dev/ paths (likely image files)
		// which might be handled by external libraries like libewf.
		if !strings.HasPrefix(devicePath, "/dev/") && (runtime.GOOS == "linux" || runtime.GOOS == "darwin") {
			// In C, this is where `fewf_init` would be called.
			// Here, we just indicate that it's an unhandled file type.
			return nil, fmt.Errorf("failed to open file/device %s: %w (check path or file type)", devicePath, err)
		}
		return nil, fmt.Errorf("failed to open device %s: %w", devicePath, err)
	}

	// At this point, the file/device is successfully opened.
	diskInfo.file = file
	diskInfo.AccessMode = finalOpenFlags

	// --- 4. Determine Device Type and Gather Information ---
	stat, statErr := file.Stat()
	if statErr != nil {
		file.Close()
		return nil, fmt.Errorf("failed to get file/device info for %s: %w", devicePath, statErr)
	}

	// Check if it's a block device using os.FileMode.
	// On Unix-like systems, os.ModeDevice indicates a device file.
	diskInfo.IsDevice = stat.Mode()&os.ModeDevice != 0

	if diskInfo.IsDevice {
		// It's a raw block device (e.g., /dev/sda on Linux, /dev/rdiskX on macOS)
		if verbose > 1 {
			fmt.Printf("file_test_availability %s is a device\n", devicePath)
		}
		if runtime.GOOS == "linux" {
			// Linux-specific: Get sector size and total size via ioctl.
			sectorSize, ioctlErr := GetSectorSizeLinux(file)
			if ioctlErr != nil {
				if verbose > 0 {
					fmt.Printf("Warning: Could not get device sector size for %s: %v. Using default %d.\n", devicePath, ioctlErr, DefaultSectorSize)
				}
				diskInfo.SectorSize = DefaultSectorSize // Fallback
			} else {
				diskInfo.SectorSize = sectorSize
			}

			realSize, ioctlErr := GetDiskSizeLinux(file)
			if ioctlErr != nil {
				if verbose > 0 {
					fmt.Printf("Warning: Could not get real disk size for %s: %v. Attempting via Seek.\n", devicePath, ioctlErr)
				}
				// Fallback to seeking to the end of the file/device for size
				realSize, ioctlErr = file.Seek(0, io.SeekEnd)
				if ioctlErr != nil {
					file.Close()
					return nil, fmt.Errorf("could not determine device size for %s: %w", devicePath, ioctlErr)
				}
			}
			diskInfo.RealSize = realSize

			// Linux-specific: Discard buffer cache (BLKFLSBUF) - similar to fdisk trick
			//syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), syscall.BLKFLSBUF, 0)
		} else {
			// For non-Linux devices, fall back to seeking for size and default sector size.
			// Proper handling for macOS/Windows devices would involve specific syscalls/APIs.
			if verbose > 0 {
				fmt.Printf("Warning: Non-Linux device detected for %s. Cannot get precise sector size via ioctl. Using default %d bytes.\n", devicePath, DefaultSectorSize)
			}
			diskInfo.RealSize, err = file.Seek(0, io.SeekEnd)
			if err != nil {
				file.Close()
				return nil, fmt.Errorf("could not determine device size via seek for %s: %w", devicePath, err)
			}
		}
		// PhotoRec also calls disk_get_geometry and disk_get_model here.
		// These are complex and OS-specific, omitted for this general translation.
	} else {
		// It's a regular file (e.g., a .dd or .img disk image)
		if verbose > 1 {
			fmt.Printf("file_test_availability %s is a regular file (disk image)\n", devicePath)
		}
		diskInfo.SectorSize = DefaultSectorSize // Assume default sector size for files

		firstBlock, readErr := ReadFirstBlock(file, diskInfo.SectorSize)
		if readErr != nil {
			if verbose > 0 {
				fmt.Printf("Warning: Could not read first block of file %s: %v. Assuming empty or truncated.\n", devicePath, readErr)
			}
			firstBlock = make([]byte, diskInfo.SectorSize) // Ensure a non-nil, possibly empty buffer
		}

		// --- Image Format Detection (Simplified: DOSEMU, EWF) ---
		// PhotoRec has detailed parsing here, potentially returning a different DiskInfo type
		// or handling via specialized readers. Here, we just check magic bytes.

		// Check for EWF signature (8 bytes)
		if len(firstBlock) >= len(evfFileSignature) && bytes.Equal(firstBlock[:len(evfFileSignature)], evfFileSignature[:]) {
			fmt.Printf("EWF format detected for %s. Full EWF parsing requires a dedicated library (e.g., CGo with libewf) and is not implemented in this example.\n", devicePath)
			diskInfo.file.Close() // Close the generic file handle, as it needs specialized handling
			return nil, fmt.Errorf("EWF format detected, specialized reader needed")
		}

		// Check for DOSEMU image magic (assuming 4 bytes)
		if len(firstBlock) >= 4 && binary.LittleEndian.Uint32(firstBlock[:4]) == DosemuImageMagic {
			fmt.Printf("DOSEMU image detected for %s. Specific parsing for geometry and offset not implemented in this example.\n", devicePath)
			// PhotoRec would parse disk_car->geom and disk_car->offset from hdr->cylinders etc.
		}

		// Get the total size of the regular file by seeking to the end.
		diskInfo.RealSize, err = file.Seek(0, io.SeekEnd)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("could not determine file size via seek for %s: %w", devicePath, err)
		}
		// For raw image files, the offset to actual data is usually 0 unless explicitly parsed (like DOSEMU).
		diskInfo.Offset = 0

		// PhotoRec also calls autoset_geometry here based on the first sector content.
		// This would involve parsing partition tables (MBR/GPT) or other metadata,
		// which is a complex topic omitted for this general translation.
	}

	// --- 5. Final Checks and Return ---
	if diskInfo.RealSize == 0 {
		file.Close()
		return nil, fmt.Errorf("failed to get a non-zero size for %s", devicePath)
	}

	return diskInfo, nil
}
