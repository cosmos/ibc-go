package types_test

import (
	"crypto/sha256"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

func (suite *TypesTestSuite) TestGetCodeHashes() {
	testCases := []struct {
		name      string
		malleate  func()
		expResult func(codeHashes [][]byte)
	}{
		{
			"success: no contract stored.",
			func() {},
			func(codeHashes [][]byte) {
				suite.Require().Len(codeHashes, 0)
			},
		},
		{
			"success: default mock vm contract stored.",
			func() {
				suite.SetupWasmWithMockVM()
			},
			func(codeHashes [][]byte) {
				suite.Require().Len(codeHashes, 1)
				expectedCodeHash := sha256.Sum256(wasmtesting.Code)
				suite.Require().Equal(expectedCodeHash[:], codeHashes[0])
			},
		},
		{
			"success: non-empty code hashes",
			func() {
				suite.SetupWasmWithMockVM()

				err := types.AddCodeHash(suite.chainA.GetContext(), []byte("codehash"))
				suite.Require().NoError(err)
			},
			func(codeHashes [][]byte) {
				suite.Require().Len(codeHashes, 2)
				suite.Require().Equal([]byte("codehash"), codeHashes[1])
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			tc.malleate()

			codeHashes, err := types.GetAllCodeHashes(suite.chainA.GetContext())
			suite.Require().NoError(err)
			tc.expResult(codeHashes)
		})
	}
}

func (suite *TypesTestSuite) TestAddCodeHash() {
	suite.SetupWasmWithMockVM()

	codeHashes, err := types.GetAllCodeHashes(suite.chainA.GetContext())
	suite.Require().NoError(err)
	// default mock vm contract is stored
	suite.Require().Len(codeHashes, 1)

	codeHash1 := []byte("codehash1")
	codeHash2 := []byte("codehash2")
	err = types.AddCodeHash(suite.chainA.GetContext(), codeHash1)
	suite.Require().NoError(err)
	err = types.AddCodeHash(suite.chainA.GetContext(), codeHash2)
	suite.Require().NoError(err)

	codeHashes, err = types.GetAllCodeHashes(suite.chainA.GetContext())
	suite.Require().NoError(err)
	suite.Require().Len(codeHashes, 3)
	suite.Require().Equal(codeHash1, codeHashes[1])
	suite.Require().Equal(codeHash2, codeHashes[2])
}

func (suite *TypesTestSuite) TestHasCodeHash() {
	var codeHash []byte

	testCases := []struct {
		name       string
		malleate   func()
		exprResult bool
	}{
		{
			"success: code hash exists",
			func() {
				codeHash = []byte("codehash")
				err := types.AddCodeHash(suite.chainA.GetContext(), codeHash)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"success: code hash does not exist",
			func() {
				codeHash = []byte("non-existent-codehash")
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			tc.malleate()

			result := types.HasCodeHash(suite.chainA.GetContext(), codeHash)
			suite.Require().Equal(tc.exprResult, result)
		})
	}
}
