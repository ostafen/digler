<p align="center">
<img alt="Digler Logo" src="assets/logo.png" width="300px">
</p>
<h2 align="center">Digler - Go Deep. Get Back Your Data</h2>

<p align="center">
  <a href="https://github.com/ostafen/digler/actions/workflows/build.yml">
    <img src="https://github.com/ostafen/digler/actions/workflows/build.yml/badge.svg" alt="Build Status">
  </a>
</p>

## Why Digler?

While many data recovery tools exist, there wasn’t yet a solution written in Go—a language well-suited for building fast, reliable, and maintainable software. Digler fills that gap by offering a simple, efficient, and cross-platform alternative focused on deep disk analysis and recovery.

## Features

* **Broad Disk Image and Raw Device Support**: Analyze a wide array of disk image formats (`.dd`, `.img`, etc...) or directly access physical disks.

* **File System Agnostic Analysis**: Recover deleted files regardless of the underlying file system (e.g., NTFS, FAT32, ext4), even when metadata is lost.

* **Post-Scan Data Recovery**: Utilize the generated DFXML reports to precisely recover deleted or fragmented files.

* **Intuitive Command-Line Interface**:  A user-friendly CLI designed for efficiency and ease of use.

* **Reporting Capabilities**: Generate detailed reports, compliant with the `Digital Forensics XML (DFXML)` format, of recovered data and analysis findings.


---

## Installation

**From Source:**

```bash
git clone https://github.com/ostafen/digler.git
cd digler
make build
```

**From Precompiled Binaries:**

Precompiled binaries will be available for Linux, macOS, and Windows on the Releases page.

## Usage

Digler follows a simple but powerful workflow: **scan first, recover later**. This approach lets you analyze disks or images thoroughly before extracting any files.

### 1. Scan a Disk Image or Device
```bash
foo@bar$ digler scan <image_or_device>
```

Example:

###
```bash
foo@bar$ digler scan dfrws-2006-challenge.raw
```

or, to scan an entire disk partition:

###
```bash
foo@bar$ digler scan /dev/nvme0n1 # or C: on Windows
```
By default, the command generates a detailed DFXML report describing the findings, together with a detailed execution log. However, you can optionally specify a dump directory to to recover files immediately during scanning.

```bash
foo@bar$ --dump <path/to/dump/dir>
```

### 2. Mount Scan Results as a Filesystem (Linux only)
```bash
foo@bar$ digler mount <image_or_device> <report_file.xml> --mountpoint /path/to/mnt
```

Example:

```bash
digler mount dfrws-2006-challenge.raw report.xml --mountpoint /mnt/recover
```

This mounts a FUSE filesystem allowing you to browse and access recovered files directly from the scan report, without copying anything yet.

### 3. Recover Files Based on Scan Report
```bash
foo@bar$ digler recover <image_or_device> <report_file.xml> --dir /path/to/dir
```

Example:

```bash
foo@bar$ digler recover dfrws-2006-challenge.raw report.xml --dir ./recover
```

### Supported File Types

Digler allows for the recovery of lost or deleted files based on their unique headers and footers, even when file system metadata is corrupted or missing. Below is a list of all currently supported file formats:

- **Documents**: docx, xlsx, pdf
- **Images**: jpg, png, gif, bmp, tiff, raw
- **Audio**: mp3, wav, flac
- **Archives**: zip

### License

Digler is released under the **MIT License**.