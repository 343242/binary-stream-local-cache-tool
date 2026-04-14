package unit

import (
	"bytes"
	"errors"
	"testing"

	"fastReadFile/internal/codec"
	"fastReadFile/pkg/cache"
)

func TestCodecRecordRoundTrip(t *testing.T) {
	t.Parallel()

	encoded, err := codec.EncodeRecord(42, cache.RawRecord{
		EventTimeUnixMs: 12345,
		Payload:         []byte("payload"),
	})
	if err != nil {
		t.Fatalf("EncodeRecord() error = %v", err)
	}

	decoded, err := codec.DecodeRecord(encoded)
	if err != nil {
		t.Fatalf("DecodeRecord() error = %v", err)
	}
	if decoded.EventTimeUnixMs != 12345 {
		t.Fatalf("EventTimeUnixMs = %d, want %d", decoded.EventTimeUnixMs, 12345)
	}
	if decoded.WriteSeq != 42 {
		t.Fatalf("WriteSeq = %d, want %d", decoded.WriteSeq, 42)
	}
	if !bytes.Equal(decoded.Payload, []byte("payload")) {
		t.Fatalf("Payload = %q, want %q", decoded.Payload, []byte("payload"))
	}
}

func TestCodecRecordRejectsPayloadCRCMismatch(t *testing.T) {
	t.Parallel()

	encoded, err := codec.EncodeRecord(7, cache.RawRecord{
		EventTimeUnixMs: 88,
		Payload:         []byte("payload"),
	})
	if err != nil {
		t.Fatalf("EncodeRecord() error = %v", err)
	}

	encoded[len(encoded)-1] ^= 0xFF
	_, err = codec.DecodeRecord(encoded)
	if !errors.Is(err, cache.ErrCode(cache.ErrCorruption)) {
		t.Fatalf("DecodeRecord() error = %v, want corruption", err)
	}
}

func TestCodecBlockRoundTrip(t *testing.T) {
	t.Parallel()

	recordOne, err := codec.EncodeRecord(10, cache.RawRecord{
		EventTimeUnixMs: 100,
		Payload:         []byte("one"),
	})
	if err != nil {
		t.Fatalf("EncodeRecord(recordOne) error = %v", err)
	}
	recordTwo, err := codec.EncodeRecord(11, cache.RawRecord{
		EventTimeUnixMs: 200,
		Payload:         []byte("two"),
	})
	if err != nil {
		t.Fatalf("EncodeRecord(recordTwo) error = %v", err)
	}

	block, err := codec.BuildBlock([][]byte{recordOne, recordTwo}, 10, 11, 100, 200)
	if err != nil {
		t.Fatalf("BuildBlock() error = %v", err)
	}

	decoded, err := codec.ParseBlock(block)
	if err != nil {
		t.Fatalf("ParseBlock() error = %v", err)
	}
	if decoded.RecordCount != 2 {
		t.Fatalf("RecordCount = %d, want %d", decoded.RecordCount, 2)
	}
	if decoded.FirstWriteSeq != 10 {
		t.Fatalf("FirstWriteSeq = %d, want %d", decoded.FirstWriteSeq, 10)
	}
	if decoded.LastWriteSeq != 11 {
		t.Fatalf("LastWriteSeq = %d, want %d", decoded.LastWriteSeq, 11)
	}
	if len(decoded.RecordBytes) != 2 {
		t.Fatalf("len(RecordBytes) = %d, want %d", len(decoded.RecordBytes), 2)
	}
	if !bytes.Equal(decoded.RecordBytes[0], recordOne) {
		t.Fatalf("first record bytes changed during block round trip")
	}
	if !bytes.Equal(decoded.RecordBytes[1], recordTwo) {
		t.Fatalf("second record bytes changed during block round trip")
	}
}
