package unit

import (
	"encoding/binary"
	"testing"

	"fastReadFile/internal/simtool"
)

func TestSampleGeneratorYieldsExpectedTopologyAndTimestamps(t *testing.T) {
	profile := simtool.Profile{
		Name:             simtool.ProfileMedium,
		Gateways:         2,
		PointsPerGateway: 3,
		Rounds:           2,
		RoundStepMs:      1000,
	}
	gen := simtool.NewBatchGenerator(profile, 4, 1_000)

	var (
		total         int
		firstPayloads [][]byte
		observedTimes []int64
	)
	for {
		batch, ok := gen.NextBatch()
		if !ok {
			break
		}
		total += len(batch)
		for _, record := range batch {
			if len(firstPayloads) < 4 {
				firstPayloads = append(firstPayloads, append([]byte(nil), record.Payload...))
			}
			observedTimes = append(observedTimes, record.EventTimeUnixMs)
		}
	}

	if total != 12 {
		t.Fatalf("total records = %d, want %d", total, 12)
	}
	if got := decodeU32(firstPayloads[0][0:4]); got != 1 {
		t.Fatalf("first gateway = %d, want %d", got, 1)
	}
	if got := decodeU32(firstPayloads[0][4:8]); got != 1 {
		t.Fatalf("first point = %d, want %d", got, 1)
	}
	if got := decodeU32(firstPayloads[3][0:4]); got != 2 {
		t.Fatalf("fourth gateway = %d, want %d", got, 2)
	}
	if got := decodeU32(firstPayloads[3][4:8]); got != 1 {
		t.Fatalf("fourth point = %d, want %d", got, 1)
	}
	if observedTimes[0] != 1_000 || observedTimes[5] != 1_000 || observedTimes[6] != 2_000 {
		t.Fatalf("observed times = %v, want round-based timestamps", observedTimes)
	}
}

func TestBuildBatchesYieldsPartialFinalBatch(t *testing.T) {
	profile := simtool.Profile{
		Name:             simtool.ProfileMedium,
		Gateways:         1,
		PointsPerGateway: 5,
		Rounds:           1,
		RoundStepMs:      1000,
	}
	gen := simtool.NewBatchGenerator(profile, 2, 1_000)

	var sizes []int
	for {
		batch, ok := gen.NextBatch()
		if !ok {
			break
		}
		sizes = append(sizes, len(batch))
	}
	if len(sizes) != 3 || sizes[0] != 2 || sizes[1] != 2 || sizes[2] != 1 {
		t.Fatalf("batch sizes = %v, want [2 2 1]", sizes)
	}
}

func decodeU32(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf)
}
