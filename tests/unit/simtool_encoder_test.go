package unit

import (
	"encoding/binary"
	"math"
	"testing"

	"fastReadFile/internal/simtool"
)

func TestEncodeSampleProducesStableFixedWidthBytes(t *testing.T) {
	sample := simtool.SensorSample{
		GatewayID:     17,
		PointID:       2048,
		Value:         12.5,
		CollectTimeMs: 1710000000123,
	}

	encoded := simtool.EncodeSample(sample)
	if len(encoded) != 24 {
		t.Fatalf("len(encoded) = %d, want %d", len(encoded), 24)
	}
	if got := binary.LittleEndian.Uint32(encoded[0:4]); got != sample.GatewayID {
		t.Fatalf("GatewayID = %d, want %d", got, sample.GatewayID)
	}
	if got := binary.LittleEndian.Uint32(encoded[4:8]); got != sample.PointID {
		t.Fatalf("PointID = %d, want %d", got, sample.PointID)
	}
	if got := math.Float64frombits(binary.LittleEndian.Uint64(encoded[8:16])); got != sample.Value {
		t.Fatalf("Value = %v, want %v", got, sample.Value)
	}
	if got := int64(binary.LittleEndian.Uint64(encoded[16:24])); got != sample.CollectTimeMs {
		t.Fatalf("CollectTimeMs = %d, want %d", got, sample.CollectTimeMs)
	}
}
