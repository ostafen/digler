package scan

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/ostafen/diglet/internal/disk"
	"github.com/ostafen/diglet/internal/format"
)

func Scan(filePath, dumpDir string) error {
	partitions, err := DiscoverPartitions(filePath)
	if err != nil {
		return err
	}

	scanAllPartitions := true
	partitionsToScan := map[int]bool{}

	for _, p := range partitions {
		if scanAllPartitions || partitionsToScan[p.Num] {
			if err := ScanPartition(&p, filePath, dumpDir); err != nil {
				return err
			}
		}
	}
	return nil
}

func ScanPartition(p *disk.Partition, filePath, dumpDir string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := io.NewSectionReader(f, int64(p.Offset), int64(p.Size))

	if dumpDir != "" {
		if err := os.MkdirAll(dumpDir, 0755); err != nil {
			return err
		}
	}

	numFiles := 0

	sc := format.NewScanner(uint64(p.BlockSize))

	for finfo := range sc.Scan(r, p.Size) {
		log.Printf("found %s file at block %d, size %d bytes\n", finfo.Format, finfo.Offset/uint64(p.BlockSize), finfo.Size)

		if dumpDir != "" {
			numFiles++

			fileName := fmt.Sprintf("%d.%s", numFiles, finfo.Format)

			fileReader := io.NewSectionReader(r, int64(finfo.Offset), int64(finfo.Size))
			err := dumpFile(dumpDir, fileName, fileReader)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func dumpFile(dumpDir string, fileName string, r io.Reader) error {
	f, err := os.Create(filepath.Join(dumpDir, fileName))
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", fileName, err)
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 1024*1024) // 1MB buffer

	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return w.Flush()
}

func DiscoverPartitions(path string) ([]disk.Partition, error) {
	imgFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file %q: %w", path, err)
	}
	defer imgFile.Close()

	var firstSector [512]byte
	_, err = imgFile.ReadAt(firstSector[:], 0)
	if err != nil {
		return nil, err
	}

	// Try to parse the first sector as an MBR
	mbr, err := disk.ParseMBR(firstSector[:])
	if err == nil {
		return GetMBRPartitions(imgFile, mbr)
	}

	// TODO: if was unable to determine the partitions,
	// Maybe, try to locate partition boundaries using FAT signature, etc?

	finfo, err := imgFile.Stat()
	if err != nil {
		return nil, err
	}

	return []disk.Partition{
		fullDiskPartition(uint64(finfo.Size())),
	}, nil
}

func fullDiskPartition(diskSize uint64) disk.Partition {
	return disk.Partition{
		FSType:    1,
		Num:       0,
		Offset:    0,
		Size:      diskSize,
		BlockSize: disk.DefaultBlocksize,
	}
}

func GetMBRPartitions(imgFile *os.File, mbr *disk.MBR) ([]disk.Partition, error) {
	partitions := make([]disk.Partition, 0, len(mbr.PartitionEntries))
	for n, p := range mbr.PartitionEntries {
		switch p.PartitionType {
		case disk.PartitionTypeFAT12,
			disk.PartitionTypeFAT16LessThan32MB,
			disk.PartitionTypeFAT16GreaterThan32MB,
			disk.PartitionTypeFAT16LBA,
			disk.PartitionTypeFAT32LBA,
			disk.PartitionTypeFAT32CHS:

			offset := int64(p.ReadStartLBA()) * disk.DefaultSectorSize

			var buf [512]byte
			_, err := imgFile.ReadAt(buf[:], offset)
			if err != nil {
				continue
			}

			fatSector, err := disk.ReadFatBootSectorFrom(buf[:])
			if err == nil {
				partitions = append(partitions, disk.Partition{
					FSType:    0,
					Num:       n,
					Offset:    uint64(offset),
					BlockSize: uint32(fatSector.SectorSize),
				})
			}
		}
	}
	return partitions, nil
}
