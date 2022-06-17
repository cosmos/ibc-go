package testconfig

import (
	"fmt"
	"os"
)

const (
	DefaultSimdImage = "ghcr.io/cosmos/ibc-go-simd"

	SimdImageEnv = "SIMD_IMAGE"
	SimdTagEnv   = "SIMD_TAG"
)

type TestConfig struct {
	SimdImage string
	SimdTag   string
}

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
