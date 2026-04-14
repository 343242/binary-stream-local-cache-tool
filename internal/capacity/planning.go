package capacity

const (
	secondsPerDay         = 24 * 60 * 60
	decimalTerabyte int64 = 1_000_000_000_000
)

type Plan struct {
	PayloadBytes           int64
	RecommendedUsableBytes int64
}

func Estimate(recordsPerSecond int64, payloadBytes int64, days int64) Plan {
	totalPayload := recordsPerSecond * secondsPerDay * days * payloadBytes
	return Plan{
		PayloadBytes:           totalPayload,
		RecommendedUsableBytes: totalPayload * 2,
	}
}

func DecimalTerabyte() int64 {
	return decimalTerabyte
}
