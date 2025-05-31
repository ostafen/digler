<p align="center">
<img alt="Digler Logo" src="assets/logo.png" width="300px">
</p>
<h2 align="center">Digler - Go Deep. Get Back Your Data</h2>

## Features

* **Comprehensive Disk Image Support**: Analyze a wide range of disk image formats, including raw (`.dd`, `.img`), EWF (`.e01`), and others.
* **Raw Device Access**: Directly access and analyze physical disks and partitions for real-time forensic investigations.
* **Advanced File Carving**: Recover deleted or fragmented files based on their unique signatures, even when file system metadata is lost.
* **File System Agnostic Analysis**: Go beyond traditional file system structures to uncover data regardless of the underlying file system (e.g., NTFS, FAT32, ext4).
* **Metadata Extraction**: Extract valuable metadata from files and file systems to aid in your investigation.
* **Intuitive User Interface**: (Assuming there will be one or if it's CLI based, describe its ease of use) A user-friendly interface designed for both novice and experienced forensicators.
* **Reporting Capabilities**: Generate detailed reports of recovered data and analysis findings.


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

### License

Digler is released under the **MIT License**.