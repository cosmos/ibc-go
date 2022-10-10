package testsuite_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/e2e/testsuite"
)

func TestIsSupported(t *testing.T) {
	releases := testsuite.FeatureReleases{
		MajorVersion: 6,
		MinorVersions: []testsuite.Version{
			testsuite.NewVersion(2, 5),
			testsuite.NewVersion(3, 4),
			testsuite.NewVersion(4, 2),
			testsuite.NewVersion(5, 1),
		},
	}

	testCases := []struct {
		name         string
		version      string
		expSupported bool
	}{
		{"non semantic version", "main", true},
		{"non semantic version starts with v", "v", true},
		{"non semantic version", "pr-155", true},
		{"non semantic version", "major.5.1", true},
		{"non semantic version", "1.minor.1", true},
		{"supported semantic version", "v2.5.0", true},
		{"supported semantic version", "v3.4.0", true},
		{"supported semantic version", "v4.2.0", true},
		{"supported semantic version", "v5.1.0", true},
		{"supported semantic version", "v6.0.0", true},
		{"supported semantic version", "v6.1.0", true},
		{"supported semantic version", "v7.1.0", true},
		{"supported semantic version", "v22.5.1", true},
		{"supported semantic version with v", "2.5.0", true},
		{"unsupported semantic version", "v1.5.0", false},
		{"unsupported semantic version", "v2.4.5", false},
		{"unsupported semantic version", "v3.1.0", false},
		{"unsupported semantic version", "v4.1.0", false},
		{"unsupported semantic version", "v5.0.0", false},
		{"unsupported semantic version on partially supported major line", "v2.4.0", false},
	}

	for _, tc := range testCases {
		supported := releases.IsSupported(tc.version)
		require.Equal(t, tc.expSupported, supported, tc.name)
	}
}
