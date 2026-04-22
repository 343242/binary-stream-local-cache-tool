package simtool

import (
	"fmt"
	"strings"
)

type ProfileName string

const (
	ProfileMedium ProfileName = "medium"
	ProfileLarge  ProfileName = "large"
)

const (
	DefaultBatchSize = 1000
	DefaultRoundStep = int64(1000)
)

type Profile struct {
	Name             ProfileName
	Gateways         int
	PointsPerGateway int
	Rounds           int
	RoundStepMs      int64
}

func ResolveProfile(name string) (Profile, error) {
	switch normalized := strings.ToLower(strings.TrimSpace(name)); normalized {
	case "", string(ProfileMedium):
		return Profile{
			Name:             ProfileMedium,
			Gateways:         20,
			PointsPerGateway: 2000,
			Rounds:           5,
			RoundStepMs:      DefaultRoundStep,
		}, nil
	case string(ProfileLarge):
		return Profile{
			Name:             ProfileLarge,
			Gateways:         50,
			PointsPerGateway: 5000,
			Rounds:           3,
			RoundStepMs:      DefaultRoundStep,
		}, nil
	default:
		return Profile{}, fmt.Errorf("unknown simulator profile %q", name)
	}
}

func normalizeProfile(profile Profile) (Profile, error) {
	if profile.Gateways == 0 && profile.PointsPerGateway == 0 && profile.Rounds == 0 {
		return ResolveProfile(string(profile.Name))
	}
	if profile.Name == "" {
		profile.Name = ProfileName("custom")
	}
	if profile.Gateways <= 0 || profile.PointsPerGateway <= 0 || profile.Rounds <= 0 {
		return Profile{}, fmt.Errorf("invalid simulator profile %+v", profile)
	}
	if profile.RoundStepMs <= 0 {
		profile.RoundStepMs = DefaultRoundStep
	}
	return profile, nil
}

func (p Profile) TotalRecords() int {
	return p.Gateways * p.PointsPerGateway * p.Rounds
}
