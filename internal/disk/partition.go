package disk

/*
struct partition_struct
{
  char          fsname[128];
  char          partname[128];
  char          info[128];
  uint64_t      part_offset;
  uint64_t      part_size;
  uint64_t      sborg_offset;
  uint64_t      sb_offset;
  unsigned int  sb_size;
  unsigned int  blocksize;
  efi_guid_t    part_uuid;
  efi_guid_t    part_type_gpt;
  unsigned int  part_type_humax;
  unsigned int  part_type_i386;
  unsigned int  part_type_mac;
  unsigned int  part_type_sun;
  unsigned int  part_type_xbox;
  upart_type_t  upart_type;
  status_type_t status;
  unsigned int  order;
  errcode_type_t errcode;
  const arch_fnct_t *arch;
};
*/

type (
	PartitionType uint8
	FSType        uint8
)

type Partition struct {
	FSType    FSType
	Num       int
	Offset    uint64 // Offset in bytes from the start of the disk
	Size      uint64 // Size in bytes of the partition
	BlockSize uint32 // Block size in bytes
}
