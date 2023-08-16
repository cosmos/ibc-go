package types

import (
	"crypto/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateWasmCode(t *testing.T) {
	var code []byte

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				code, _ = os.ReadFile("../test_data/ics10_grandpa_cw.wasm.gz")
			},
			true,
		},
		{
			"fails with empty byte slice",
			func() {
				code = []byte{}
			},
			false,
		},
		{
			"fails with byte slice too large",
			func() {
				expLength := MaxWasmByteSize() + 1
				code = make([]byte, expLength)
				length, err := rand.Read(code)
				require.NoError(t, err, t.Name())
				require.Equal(t, expLength, uint64(length), t.Name())
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc.malleate()

		err := ValidateWasmCode(code)

		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}
