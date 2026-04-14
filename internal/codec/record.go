package codec

import (
	"encoding/binary"
	"hash/crc32"

	cache "fastReadFile/internal/core"
)

const (
	recordMagic      uint16 = 0x5243
	recordVersion    uint16 = 1
	recordHeaderSize        = 32
)

type DecodedRecord struct {
	EventTimeUnixMs int64
	WriteSeq        uint64
	PayloadLen      uint32
	Payload         []byte
	EncodedLen      int
}

func EncodeRecord(writeSeq uint64, input cache.RawRecord) ([]byte, error) {
	if len(input.Payload) == 0 {
		return nil, cache.NewError(cache.ErrValidation, "encode_record", "", "empty payload", nil)
	}

	buf := make([]byte, recordHeaderSize+len(input.Payload))
	binary.LittleEndian.PutUint16(buf[0:2], recordMagic)
	binary.LittleEndian.PutUint16(buf[2:4], recordVersion)
	binary.LittleEndian.PutUint64(buf[4:12], uint64(input.EventTimeUnixMs))
	binary.LittleEndian.PutUint64(buf[12:20], writeSeq)
	binary.LittleEndian.PutUint32(buf[20:24], uint32(len(input.Payload)))
	binary.LittleEndian.PutUint32(buf[24:28], crc32.ChecksumIEEE(buf[0:24]))
	binary.LittleEndian.PutUint32(buf[28:32], crc32.ChecksumIEEE(input.Payload))
	copy(buf[recordHeaderSize:], input.Payload)
	return buf, nil
}

func DecodeRecord(buf []byte) (DecodedRecord, error) {
	recordLen, err := encodedRecordLength(buf)
	if err != nil {
		return DecodedRecord{}, err
	}
	if binary.LittleEndian.Uint16(buf[0:2]) != recordMagic {
		return DecodedRecord{}, cache.NewError(cache.ErrCorruption, "decode_record", "", "record magic mismatch", nil)
	}
	if binary.LittleEndian.Uint16(buf[2:4]) != recordVersion {
		return DecodedRecord{}, cache.NewError(cache.ErrCorruption, "decode_record", "", "record version mismatch", nil)
	}
	headerCRC := binary.LittleEndian.Uint32(buf[24:28])
	if headerCRC != crc32.ChecksumIEEE(buf[0:24]) {
		return DecodedRecord{}, cache.NewError(cache.ErrCorruption, "decode_record", "", "record header crc mismatch", nil)
	}

	payload := append([]byte(nil), buf[recordHeaderSize:recordLen]...)
	payloadCRC := binary.LittleEndian.Uint32(buf[28:32])
	if payloadCRC != crc32.ChecksumIEEE(payload) {
		return DecodedRecord{}, cache.NewError(cache.ErrCorruption, "decode_record", "", "record payload crc mismatch", nil)
	}

	return DecodedRecord{
		EventTimeUnixMs: int64(binary.LittleEndian.Uint64(buf[4:12])),
		WriteSeq:        binary.LittleEndian.Uint64(buf[12:20]),
		PayloadLen:      binary.LittleEndian.Uint32(buf[20:24]),
		Payload:         payload,
		EncodedLen:      recordLen,
	}, nil
}

func encodedRecordLength(buf []byte) (int, error) {
	if len(buf) < recordHeaderSize {
		return 0, cache.NewError(cache.ErrCorruption, "decode_record", "", "record too short", nil)
	}
	payloadLen := int(binary.LittleEndian.Uint32(buf[20:24]))
	recordLen := recordHeaderSize + payloadLen
	if len(buf) < recordLen {
		return 0, cache.NewError(cache.ErrCorruption, "decode_record", "", "record truncated", nil)
	}
	return recordLen, nil
}
