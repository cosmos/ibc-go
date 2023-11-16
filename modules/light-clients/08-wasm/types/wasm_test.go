package types_test

import (
	"crypto/sha256"

	wasmvm "github.com/CosmWasm/wasmvm"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/internal/ibcwasm"
	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

func (suite *TypesTestSuite) TestGetChecksums() {
	testCases := []struct {
		name      string
		malleate  func()
		expResult func(checksums []wasmvm.Checksum)
	}{
		{
			"success: no contract stored.",
			func() {},
			func(checksums []wasmvm.Checksum) {
				suite.Require().Len(checksums, 0)
			},
		},
		{
			"success: default mock vm contract stored.",
			func() {
				suite.SetupWasmWithMockVM()
			},
			func(checksums []wasmvm.Checksum) {
				suite.Require().Len(checksums, 1)
				expectedChecksum := sha256.Sum256(wasmtesting.Code)
				suite.Require().Equal(wasmvm.Checksum(expectedChecksum[:]), checksums[0])
			},
		},
		{
			"success: non-empty checksums",
			func() {
				suite.SetupWasmWithMockVM()

				err := ibcwasm.Checksums.Set(suite.chainA.GetContext(), wasmvm.Checksum("checksum"))
				suite.Require().NoError(err)
			},
			func(checksums []wasmvm.Checksum) {
				suite.Require().Len(checksums, 2)
				suite.Require().Contains(checksums, wasmvm.Checksum("checksum"))
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

	checksum1 := wasmvm.Checksum("checksum1")
	checksum2 := wasmvm.Checksum("checksum2")
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
	var checksum wasmvm.Checksum

	testCases := []struct {
		name       string
		malleate   func()
		exprResult bool
	}{
		{
			"success: checksum exists",
			func() {
				checksum = wasmvm.Checksum("checksum")
				err := ibcwasm.Checksums.Set(suite.chainA.GetContext(), checksum)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"success: checksum does not exist",
			func() {
				checksum = wasmvm.Checksum("non-existent-checksum")
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
