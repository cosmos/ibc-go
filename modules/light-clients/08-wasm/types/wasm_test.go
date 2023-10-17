package types_test

import "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"

func (suite *TypesTestSuite) TestGetCodeHashes() {
	testCases := []struct {
		name      string
		malleate  func()
		expResult func(codeHashes types.CodeHashes)
	}{
		{
			"success: empty code hashes",
			func() {},
			func(codeHashes types.CodeHashes) {
				suite.Require().Empty(codeHashes.Hashes)
			},
		},
		{
			"success: non-empty code hashes",
			func() {
				types.AddCodeHash(suite.chainA.GetContext(), GetSimApp(suite.chainA).AppCodec(), []byte("codehash"))
			},
			func(codeHashes types.CodeHashes) {
				suite.Require().Len(codeHashes.Hashes, 1)
				suite.Require().Equal([]byte("codehash"), codeHashes.Hashes[0])
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			tc.malleate()

			codeHashes := types.GetCodeHashes(suite.chainA.GetContext(), GetSimApp(suite.chainA).AppCodec())
			tc.expResult(codeHashes)
		})
	}
}

func (suite *TypesTestSuite) TestAddCodeHash() {
	suite.SetupWasmWithMockVM()

	codeHashes := types.GetCodeHashes(suite.chainA.GetContext(), GetSimApp(suite.chainA).AppCodec())
	suite.Require().Empty(codeHashes.Hashes)

	codeHash1 := []byte("codehash1")
	codeHash2 := []byte("codehash2")
	types.AddCodeHash(suite.chainA.GetContext(), GetSimApp(suite.chainA).AppCodec(), codeHash1)
	types.AddCodeHash(suite.chainA.GetContext(), GetSimApp(suite.chainA).AppCodec(), codeHash2)

	codeHashes = types.GetCodeHashes(suite.chainA.GetContext(), GetSimApp(suite.chainA).AppCodec())
	suite.Require().Len(codeHashes.Hashes, 2)
	suite.Require().Equal(codeHash1, codeHashes.Hashes[0])
	suite.Require().Equal(codeHash2, codeHashes.Hashes[1])
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
				types.AddCodeHash(suite.chainA.GetContext(), GetSimApp(suite.chainA).AppCodec(), codeHash)
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

			result := types.HasCodeHash(suite.chainA.GetContext(), GetSimApp(suite.chainA).AppCodec(), codeHash)
			suite.Require().Equal(tc.exprResult, result)
		})
	}
}
