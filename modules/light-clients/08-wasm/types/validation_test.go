package types_test

import (
	"crypto/rand"
	"crypto/sha256"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

func TestValidateWasmCode(t *testing.T) {
	var code []byte

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				code, _ = os.ReadFile("../test_data/ics10_grandpa_cw.wasm.gz")
			},
			nil,
		},
		{
			"failure: empty byte slice",
			func() {
				code = []byte{}
			},
			types.ErrWasmEmptyCode,
		},
		{
			"failure: byte slice too large",
			func() {
				expLength := types.MaxWasmByteSize() + 1
				code = make([]byte, expLength)
				length, err := rand.Read(code)
				require.NoError(t, err, t.Name())
				require.Equal(t, expLength, uint64(length), t.Name())
			},
			types.ErrWasmCodeTooLarge,
		},
	}

	for _, tc := range testCases {
		tc.malleate()

		err := types.ValidateWasmCode(code)

		if tc.expError == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.ErrorIs(t, err, tc.expError, tc.name)
		}
	}
}

func TestValidateWasmCodeHash(t *testing.T) {
	var codeHash []byte

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				code, _ := os.ReadFile("../test_data/ics10_grandpa_cw.wasm.gz")
				checksum := sha256.Sum256(code)
				codeHash = checksum[:]
			},
			nil,
		},
		{
			"failure: nil byte slice",
			func() {
				codeHash = nil
			},
			errorsmod.Wrap(types.ErrInvalidCodeHash, "code hash cannot be empty"),
		},
		{
			"failure: empty byte slice",
			func() {
				codeHash = []byte{}
			},
			errorsmod.Wrap(types.ErrInvalidCodeHash, "code hash cannot be empty"),
		},
		{
			"failure: byte slice size is not 32",
			func() {
				codeHash = []byte{1}
			},
			errorsmod.Wrapf(types.ErrInvalidCodeHash, "expected length of 32 bytes, got %d", 1),
		},
	}

	for _, tc := range testCases {
		tc.malleate()

		err := types.ValidateWasmCodeHash(codeHash)

		if tc.expError == nil {
			require.NoError(t, err, tc.name)
		} else {
			require.ErrorIs(t, err, tc.expError, tc.name)
		}
	}
}
