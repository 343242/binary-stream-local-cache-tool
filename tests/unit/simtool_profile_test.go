package unit

import (
	"testing"

	"fastReadFile/internal/simtool"
)

func TestSimProfileDefaultsToMedium(t *testing.T) {
	profile, err := simtool.ResolveProfile("")
	if err != nil {
		t.Fatalf("ResolveProfile(\"\") error = %v", err)
	}
	if profile.Name != simtool.ProfileMedium {
		t.Fatalf("Name = %q, want %q", profile.Name, simtool.ProfileMedium)
	}
	if profile.Gateways != 20 || profile.PointsPerGateway != 2000 || profile.Rounds != 5 {
		t.Fatalf("profile = %+v, want medium defaults", profile)
	}
}

func TestSimProfileResolvesLarge(t *testing.T) {
	profile, err := simtool.ResolveProfile("large")
	if err != nil {
		t.Fatalf("ResolveProfile(\"large\") error = %v", err)
	}
	if profile.Name != simtool.ProfileLarge {
		t.Fatalf("Name = %q, want %q", profile.Name, simtool.ProfileLarge)
	}
	if profile.Gateways != 50 || profile.PointsPerGateway != 5000 || profile.Rounds != 3 {
		t.Fatalf("profile = %+v, want large defaults", profile)
	}
}

func TestSimProfileRejectsUnknownProfile(t *testing.T) {
	if _, err := simtool.ResolveProfile("xlarge"); err == nil {
		t.Fatal("ResolveProfile(\"xlarge\") unexpectedly succeeded")
	}
}
