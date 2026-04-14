package segment

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	cache "fastReadFile/internal/core"
)

const (
	footerMagic   uint32 = 0x53464754
	footerVersion uint16 = 1
	FooterSize           = 96
)

type Footer struct {
	Version         uint16
	SegmentID       uint64
	CreatedAtUnixMs int64
	SealedAtUnixMs  int64
	DataEndOffset   uint64
	RecordCount     uint64
	FirstWriteSeq   uint64
	LastWriteSeq    uint64
	MinEventTime    int64
	MaxEventTime    int64
	LastBatchSeq    uint64
	CRC32           uint32
}

func EncodeFooter(footer Footer) ([]byte, error) {
	if footer.SegmentID == 0 {
		return nil, cache.NewError(cache.ErrValidation, "encode_footer", "", "segment id must be positive", nil)
	}
	buf := make([]byte, FooterSize)
	binary.LittleEndian.PutUint32(buf[0:4], footerMagic)
	binary.LittleEndian.PutUint16(buf[4:6], footerVersion)
	binary.LittleEndian.PutUint16(buf[6:8], 0)
	binary.LittleEndian.PutUint32(buf[8:12], FooterSize)
	binary.LittleEndian.PutUint64(buf[12:20], footer.SegmentID)
	binary.LittleEndian.PutUint64(buf[20:28], uint64(footer.CreatedAtUnixMs))
	binary.LittleEndian.PutUint64(buf[28:36], uint64(footer.SealedAtUnixMs))
	binary.LittleEndian.PutUint64(buf[36:44], footer.DataEndOffset)
	binary.LittleEndian.PutUint64(buf[44:52], footer.RecordCount)
	binary.LittleEndian.PutUint64(buf[52:60], footer.FirstWriteSeq)
	binary.LittleEndian.PutUint64(buf[60:68], footer.LastWriteSeq)
	binary.LittleEndian.PutUint64(buf[68:76], uint64(footer.MinEventTime))
	binary.LittleEndian.PutUint64(buf[76:84], uint64(footer.MaxEventTime))
	binary.LittleEndian.PutUint64(buf[84:92], footer.LastBatchSeq)
	binary.LittleEndian.PutUint32(buf[92:96], crc32.ChecksumIEEE(buf[:92]))
	return buf, nil
}

func ReadFooter(path string) (Footer, error) {
	file, err := os.Open(path)
	if err != nil {
		return Footer{}, cache.NewError(cache.ErrIO, "read_footer", path, "open segment file", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return Footer{}, cache.NewError(cache.ErrIO, "read_footer", path, "stat segment file", err)
	}
	if stat.Size() < FooterSize {
		return Footer{}, cache.NewError(cache.ErrSegmentFooterInvalid, "read_footer", path, "segment too short for footer", nil)
	}

	buf := make([]byte, FooterSize)
	if _, err := file.ReadAt(buf, stat.Size()-FooterSize); err != nil {
		return Footer{}, cache.NewError(cache.ErrIO, "read_footer", path, "read segment footer", err)
	}

	footer, err := DecodeFooter(buf)
	if err != nil {
		return Footer{}, err
	}
	if err := ValidateFooter(path, stat.Size(), footer); err != nil {
		return Footer{}, err
	}
	return footer, nil
}

func DecodeFooter(buf []byte) (Footer, error) {
	if len(buf) != FooterSize {
		return Footer{}, cache.NewError(cache.ErrSegmentFooterInvalid, "decode_footer", "", "footer length invalid", nil)
	}
	if binary.LittleEndian.Uint32(buf[0:4]) != footerMagic {
		return Footer{}, cache.NewError(cache.ErrSegmentFooterInvalid, "decode_footer", "", "footer magic mismatch", nil)
	}
	if binary.LittleEndian.Uint16(buf[4:6]) != footerVersion {
		return Footer{}, cache.NewError(cache.ErrSegmentFooterInvalid, "decode_footer", "", "footer version mismatch", nil)
	}
	if binary.LittleEndian.Uint32(buf[8:12]) != FooterSize {
		return Footer{}, cache.NewError(cache.ErrSegmentFooterInvalid, "decode_footer", "", "footer size invalid", nil)
	}
	if binary.LittleEndian.Uint32(buf[92:96]) != crc32.ChecksumIEEE(buf[:92]) {
		return Footer{}, cache.NewError(cache.ErrSegmentFooterInvalid, "decode_footer", "", "footer crc mismatch", nil)
	}
	return Footer{
		Version:         binary.LittleEndian.Uint16(buf[4:6]),
		SegmentID:       binary.LittleEndian.Uint64(buf[12:20]),
		CreatedAtUnixMs: int64(binary.LittleEndian.Uint64(buf[20:28])),
		SealedAtUnixMs:  int64(binary.LittleEndian.Uint64(buf[28:36])),
		DataEndOffset:   binary.LittleEndian.Uint64(buf[36:44]),
		RecordCount:     binary.LittleEndian.Uint64(buf[44:52]),
		FirstWriteSeq:   binary.LittleEndian.Uint64(buf[52:60]),
		LastWriteSeq:    binary.LittleEndian.Uint64(buf[60:68]),
		MinEventTime:    int64(binary.LittleEndian.Uint64(buf[68:76])),
		MaxEventTime:    int64(binary.LittleEndian.Uint64(buf[76:84])),
		LastBatchSeq:    binary.LittleEndian.Uint64(buf[84:92]),
		CRC32:           binary.LittleEndian.Uint32(buf[92:96]),
	}, nil
}

func ValidateFooter(path string, fileSize int64, footer Footer) error {
	expectedID, err := segmentIDFromPath(path)
	if err != nil {
		return cache.NewError(cache.ErrSegmentFooterInvalid, "validate_footer", path, "parse segment id from path", err)
	}
	switch {
	case footer.SegmentID != expectedID:
		return cache.NewError(cache.ErrSegmentFooterInvalid, "validate_footer", path, "footer segment id mismatch", nil)
	case footer.FirstWriteSeq > footer.LastWriteSeq:
		return cache.NewError(cache.ErrSegmentFooterInvalid, "validate_footer", path, "first write seq exceeds last write seq", nil)
	case footer.MinEventTime > footer.MaxEventTime:
		return cache.NewError(cache.ErrSegmentFooterInvalid, "validate_footer", path, "min event time exceeds max event time", nil)
	case footer.DataEndOffset > uint64(fileSize-FooterSize):
		return cache.NewError(cache.ErrSegmentFooterInvalid, "validate_footer", path, "footer data offset exceeds file size", nil)
	}
	return nil
}

func segmentIDFromPath(path string) (uint64, error) {
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	id, err := strconv.ParseUint(name, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse segment id %q: %w", name, err)
	}
	return id, nil
}
