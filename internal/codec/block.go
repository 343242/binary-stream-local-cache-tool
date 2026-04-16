package codec

import (
	"encoding/binary"
	"hash/crc32"

	cache "fastReadFile/internal/core"
)

const (
	blockMagic                  uint32 = 0x424C434B
	blockVersion                uint16 = 1
	BlockHeaderSize                    = 48
	defaultBlockTargetSizeBytes        = 1 << 20
)

type DecodedBlock struct {
	RecordCount   uint16
	BodyLen       uint32
	FirstWriteSeq uint64
	LastWriteSeq  uint64
	MinEventTime  int64
	MaxEventTime  int64
	RecordBytes   [][]byte
}

func BuildBlock(records [][]byte, firstSeq, lastSeq uint64, minEventTime, maxEventTime int64) ([]byte, error) {
	if len(records) == 0 {
		return nil, cache.NewError(cache.ErrValidation, "build_block", "", "empty block", nil)
	}
	if len(records) > 0xFFFF {
		return nil, cache.NewError(cache.ErrValidation, "build_block", "", "record count exceeds uint16", nil)
	}

	bodyLen := 0
	for _, record := range records {
		if _, err := DecodeRecord(record); err != nil {
			return nil, err
		}
		bodyLen += len(record)
	}
	if len(records) > 1 && bodyLen > defaultBlockTargetSizeBytes {
		return nil, cache.NewError(cache.ErrValidation, "build_block", "", "block body exceeds default target", nil)
	}

	buf := make([]byte, BlockHeaderSize+bodyLen)
	binary.LittleEndian.PutUint32(buf[0:4], blockMagic)
	binary.LittleEndian.PutUint16(buf[4:6], blockVersion)
	binary.LittleEndian.PutUint16(buf[6:8], uint16(len(records)))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(bodyLen))
	binary.LittleEndian.PutUint64(buf[12:20], firstSeq)
	binary.LittleEndian.PutUint64(buf[20:28], lastSeq)
	binary.LittleEndian.PutUint64(buf[28:36], uint64(minEventTime))
	binary.LittleEndian.PutUint64(buf[36:44], uint64(maxEventTime))

	offset := BlockHeaderSize
	for _, record := range records {
		copy(buf[offset:], record)
		offset += len(record)
	}

	binary.LittleEndian.PutUint32(buf[44:48], blockChecksum(buf))
	return buf, nil
}

func ParseBlock(buf []byte) (DecodedBlock, error) {
	if len(buf) < BlockHeaderSize {
		return DecodedBlock{}, cache.NewError(cache.ErrCorruption, "parse_block", "", "block too short", nil)
	}
	if binary.LittleEndian.Uint32(buf[0:4]) != blockMagic {
		return DecodedBlock{}, cache.NewError(cache.ErrCorruption, "parse_block", "", "block magic mismatch", nil)
	}
	if binary.LittleEndian.Uint16(buf[4:6]) != blockVersion {
		return DecodedBlock{}, cache.NewError(cache.ErrCorruption, "parse_block", "", "block version mismatch", nil)
	}
	bodyLen := int(binary.LittleEndian.Uint32(buf[8:12]))
	if len(buf) != BlockHeaderSize+bodyLen {
		return DecodedBlock{}, cache.NewError(cache.ErrCorruption, "parse_block", "", "block length mismatch", nil)
	}
	blockCRC := binary.LittleEndian.Uint32(buf[44:48])
	if blockCRC != blockChecksum(buf) {
		return DecodedBlock{}, cache.NewError(cache.ErrCorruption, "parse_block", "", "block crc mismatch", nil)
	}

	recordBytes := make([][]byte, 0, binary.LittleEndian.Uint16(buf[6:8]))
	offset := BlockHeaderSize
	for offset < len(buf) {
		recordLen, err := encodedRecordLength(buf[offset:])
		if err != nil {
			return DecodedBlock{}, err
		}
		recordBytes = append(recordBytes, append([]byte(nil), buf[offset:offset+recordLen]...))
		offset += recordLen
	}
	if offset != len(buf) {
		return DecodedBlock{}, cache.NewError(cache.ErrCorruption, "parse_block", "", "block record boundary mismatch", nil)
	}
	if len(recordBytes) != int(binary.LittleEndian.Uint16(buf[6:8])) {
		return DecodedBlock{}, cache.NewError(cache.ErrCorruption, "parse_block", "", "block record count mismatch", nil)
	}

	return DecodedBlock{
		RecordCount:   binary.LittleEndian.Uint16(buf[6:8]),
		BodyLen:       uint32(bodyLen),
		FirstWriteSeq: binary.LittleEndian.Uint64(buf[12:20]),
		LastWriteSeq:  binary.LittleEndian.Uint64(buf[20:28]),
		MinEventTime:  int64(binary.LittleEndian.Uint64(buf[28:36])),
		MaxEventTime:  int64(binary.LittleEndian.Uint64(buf[36:44])),
		RecordBytes:   recordBytes,
	}, nil
}

func BlockLength(buf []byte) (int, error) {
	if len(buf) < BlockHeaderSize {
		return 0, cache.NewError(cache.ErrCorruption, "block_length", "", "block too short", nil)
	}
	if binary.LittleEndian.Uint32(buf[0:4]) != blockMagic {
		return 0, cache.NewError(cache.ErrCorruption, "block_length", "", "block magic mismatch", nil)
	}
	if binary.LittleEndian.Uint16(buf[4:6]) != blockVersion {
		return 0, cache.NewError(cache.ErrCorruption, "block_length", "", "block version mismatch", nil)
	}
	bodyLen := int(binary.LittleEndian.Uint32(buf[8:12]))
	if len(buf) < BlockHeaderSize+bodyLen {
		return 0, cache.NewError(cache.ErrCorruption, "block_length", "", "block truncated", nil)
	}
	return BlockHeaderSize + bodyLen, nil
}

func blockChecksum(buf []byte) uint32 {
	hashInput := make([]byte, 0, len(buf)-4)
	hashInput = append(hashInput, buf[:44]...)
	hashInput = append(hashInput, buf[48:]...)
	return crc32.ChecksumIEEE(hashInput)
}
