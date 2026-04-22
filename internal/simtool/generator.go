package simtool

import "fastReadFile/pkg/cache"

type BatchGenerator struct {
	profile    Profile
	batchSize  int
	startTime  int64
	nextIndex  int
	totalCount int
}

func NewBatchGenerator(profile Profile, batchSize int, startTimeMs int64) *BatchGenerator {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &BatchGenerator{
		profile:    profile,
		batchSize:  batchSize,
		startTime:  startTimeMs,
		totalCount: profile.TotalRecords(),
	}
}

func (g *BatchGenerator) NextBatch() ([]cache.RawRecord, bool) {
	if g.nextIndex >= g.totalCount {
		return nil, false
	}
	remaining := g.totalCount - g.nextIndex
	size := g.batchSize
	if remaining < size {
		size = remaining
	}
	batch := make([]cache.RawRecord, 0, size)
	for len(batch) < size {
		globalIndex := g.nextIndex
		round := globalIndex / (g.profile.Gateways * g.profile.PointsPerGateway)
		indexWithinRound := globalIndex % (g.profile.Gateways * g.profile.PointsPerGateway)
		gateway := indexWithinRound/g.profile.PointsPerGateway + 1
		point := indexWithinRound%g.profile.PointsPerGateway + 1
		collectTimeMs := g.startTime + int64(round)*g.profile.RoundStepMs
		sample := SensorSample{
			GatewayID:     uint32(gateway),
			PointID:       uint32(point),
			Value:         deterministicValue(round, gateway, point),
			CollectTimeMs: collectTimeMs,
		}
		batch = append(batch, cache.RawRecord{
			EventTimeUnixMs: collectTimeMs,
			Payload:         EncodeSample(sample),
		})
		g.nextIndex++
	}
	return batch, true
}

func deterministicValue(round, gateway, point int) float64 {
	return float64(round*1_000_000+gateway*10_000+point) / 100.0
}
