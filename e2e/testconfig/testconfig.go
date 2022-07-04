package testconfig

import (
	"fmt"
	"os"
)

const (
	DefaultSimdImage = "ghcr.io/cosmos/ibc-go-simd-e2e"
	SimdImageEnv     = "SIMD_IMAGE"
	SimdTagEnv       = "SIMD_TAG"
)

// TestConfig holds various fields used in the E2E tests.
type TestConfig struct {
	SimdImage string
	SimdTag   string
}

// FromEnv returns a TestConfig constructed from environment variables.
func FromEnv() TestConfig {
	simdImage, ok := os.LookupEnv(SimdImageEnv)
	if !ok {
		simdImage = DefaultSimdImage
	}

	simdTag, ok := os.LookupEnv(SimdTagEnv)
	if !ok {
		panic(fmt.Sprintf("must specify simd version for test with environment variable [%s]", SimdTagEnv))
	}

	return TestConfig{
		SimdImage: simdImage,
		SimdTag:   simdTag,
	}
}
