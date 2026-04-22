package simtool

import (
	"encoding/binary"
	"math"
)

const PayloadBytes = 24

type SensorSample struct {
	GatewayID     uint32
	PointID       uint32
	Value         float64
	CollectTimeMs int64
}

func EncodeSample(sample SensorSample) []byte {
	buf := make([]byte, PayloadBytes)
	binary.LittleEndian.PutUint32(buf[0:4], sample.GatewayID)
	binary.LittleEndian.PutUint32(buf[4:8], sample.PointID)
	binary.LittleEndian.PutUint64(buf[8:16], math.Float64bits(sample.Value))
	binary.LittleEndian.PutUint64(buf[16:24], uint64(sample.CollectTimeMs))
	return buf
}
