package types_test

import (
	"crypto/sha256"
	"os"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

func (suite *TypesTestSuite) TestCodeHashKey() {
	testCases := []struct {
		name     string
		wasmfile string
	}{
		{
			"Tendermint wasm client",
			"../test_data/ics07_tendermint_cw.wasm.gz",
		},
		{
			"Grandpa wasm client",
			"../test_data/ics10_grandpa_cw.wasm.gz",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			code, _ := os.ReadFile(tc.wasmfile)

			expectedHash := generateWasmCodeHash(code)
			codeHashKey := types.CodeHashKey(expectedHash)

			suite.Equal(len(codeHashKey), types.AbsoluteCodePositionLen)

		})
	}
}

func generateWasmCodeHash(code []byte) []byte {
	hash := sha256.Sum256(code)
	return hash[:]
}
