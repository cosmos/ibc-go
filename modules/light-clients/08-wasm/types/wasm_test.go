package types_test

import (
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

func (suite *TypesTestSuite) TestGetChecksums() {
	testCases := []struct {
		name      string
		malleate  func()
		expResult func(checksums []types.Checksum)
	}{
		{
			"success: no contract stored.",
			func() {},
			func(checksums []types.Checksum) {
				suite.Require().Len(checksums, 0)
			},
		},
		{
			"success: default mock vm contract stored.",
			func() {
				suite.SetupWasmWithMockVM()
			},
			func(checksums []types.Checksum) {
				suite.Require().Len(checksums, 1)
				expectedChecksum, err := types.CreateChecksum(wasmtesting.Code)
				suite.Require().NoError(err)
				suite.Require().Equal(expectedChecksum, checksums[0])
			},
		},
		{
			"success: non-empty checksums",
			func() {
				suite.SetupWasmWithMockVM()

				err := ibcwasm.Checksums.Set(suite.chainA.GetContext(), types.Checksum("checksum"))
				suite.Require().NoError(err)
			},
			func(checksums []types.Checksum) {
				suite.Require().Len(checksums, 2)
				suite.Require().Contains(checksums, types.Checksum("checksum"))
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			tc.malleate()

			checksums, err := types.GetAllChecksums(suite.chainA.GetContext())
			suite.Require().NoError(err)
			tc.expResult(checksums)
		})
	}
}

func (suite *TypesTestSuite) TestAddChecksum() {
	suite.SetupWasmWithMockVM()

	checksums, err := types.GetAllChecksums(suite.chainA.GetContext())
	suite.Require().NoError(err)
	// default mock vm contract is stored
	suite.Require().Len(checksums, 1)

	checksum1 := types.Checksum("checksum1")
	checksum2 := types.Checksum("checksum2")
	err = ibcwasm.Checksums.Set(suite.chainA.GetContext(), checksum1)
	suite.Require().NoError(err)
	err = ibcwasm.Checksums.Set(suite.chainA.GetContext(), checksum2)
	suite.Require().NoError(err)

	// Test adding the same checksum twice
	err = ibcwasm.Checksums.Set(suite.chainA.GetContext(), checksum1)
	suite.Require().NoError(err)

	checksums, err = types.GetAllChecksums(suite.chainA.GetContext())
	suite.Require().NoError(err)
	suite.Require().Len(checksums, 3)
	suite.Require().Contains(checksums, checksum1)
	suite.Require().Contains(checksums, checksum2)
}

func (suite *TypesTestSuite) TestHasChecksum() {
	var checksum types.Checksum

	testCases := []struct {
		name       string
		malleate   func()
		exprResult bool
	}{
		{
			"success: checksum exists",
			func() {
				checksum = types.Checksum("checksum")
				err := ibcwasm.Checksums.Set(suite.chainA.GetContext(), checksum)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"success: checksum does not exist",
			func() {
				checksum = types.Checksum("non-existent-checksum")
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			tc.malleate()

			result := types.HasChecksum(suite.chainA.GetContext(), checksum)
			suite.Require().Equal(tc.exprResult, result)
		})
	}
}
