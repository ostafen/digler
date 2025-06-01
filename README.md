<p align="center">
<img alt="Digler Logo" src="assets/logo.png" width="300px">
</p>
<h2 align="center">Digler - Go Deep. Get Back Your Data</h2>

## Features

* **Broad Disk Image and Raw Device Support**: Analyze a wide array of disk image formats (`.dd`, `.img`, etc...) or directly access physical disks.

* **File System Agnostic Analysis**: Recover deleted files regardless of the underlying file system (e.g., NTFS, FAT32, ext4), even when metadata is lost.

* **Intuitive Command-Line Interface**:  A user-friendly CLI designed for efficiency and ease of use.

* **Reporting Capabilities**: Generate detailed reports, compliant with the `Digital Forensics XML (DFXML)` format, of recovered data and analysis findings.


---

## Getting Started

### Installation

**From Source:**

```bash
git clone https://github.com/ostafen/digler.git
cd digler
make build
```

**From Precompiled Binaries:**

Precompiled binaries will be available for Linux, macOS, and Windows on the Releases page.

### Supported File Types

Digler allows for the recovery of lost or deleted files based on their unique headers and footers, even when file system metadata is corrupted or missing. Below is a list of all currently supported file formats:

- **Documents**: docx, xlsx, pdf
- **Images**: jpg, png, gif, bmp, tiff, raw
- **Audio**: mp3, wav, flac
- **Archives**: zip

### License

Digler is released under the **MIT License**.