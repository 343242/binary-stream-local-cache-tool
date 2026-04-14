package unit

import (
	"testing"

	"fastReadFile/internal/capacity"
)

func TestThirtyDayCapacityPlanningMatchesDesignAssumptions(t *testing.T) {
	average := capacity.Estimate(4000, 50, 30)
	peak := capacity.Estimate(10000, 50, 30)

	if average.PayloadBytes != 518_400_000_000 {
		t.Fatalf("average PayloadBytes = %d, want %d", average.PayloadBytes, int64(518_400_000_000))
	}
	if peak.PayloadBytes != 1_296_000_000_000 {
		t.Fatalf("peak PayloadBytes = %d, want %d", peak.PayloadBytes, int64(1_296_000_000_000))
	}
	if average.RecommendedUsableBytes < capacity.DecimalTerabyte() {
		t.Fatalf("average RecommendedUsableBytes = %d, want >= %d", average.RecommendedUsableBytes, capacity.DecimalTerabyte())
	}
	if peak.RecommendedUsableBytes < 2_500_000_000_000 {
		t.Fatalf("peak RecommendedUsableBytes = %d, want >= %d", peak.RecommendedUsableBytes, int64(2_500_000_000_000))
	}
}
