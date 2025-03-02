package keeper_test

import (
	"encoding/hex"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
)

func (suite *KeeperTestSuite) TestQueryCode() {
	var req *types.QueryCodeRequest

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
				msg := types.NewMsgStoreCode(signer, wasmtesting.Code)

				res, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				req = &types.QueryCodeRequest{Checksum: hex.EncodeToString(res.Checksum)}
			},
			nil,
		},
		{
			"fails with empty request",
			func() {
				req = &types.QueryCodeRequest{}
			},
			status.Error(
				codes.NotFound,
				errorsmod.Wrap(types.ErrWasmChecksumNotFound, "").Error(),
			),
		},
		{
			"fails with non-existent checksum",
			func() {
				req = &types.QueryCodeRequest{Checksum: "test"}
			},
			status.Error(
				codes.InvalidArgument,
				types.ErrInvalidChecksum.Error(),
			),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			tc.malleate()

			res, err := GetSimApp(suite.chainA).WasmClientKeeper.Code(suite.chainA.GetContext(), req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().NotEmpty(res.Data)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestQueryChecksums() {
	var expChecksums []string

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success with no checksums",
			func() {
				expChecksums = []string{}
			},
			nil,
		},
		{
			"success with one checksum",
			func() {
				signer := authtypes.NewModuleAddress(govtypes.ModuleName).String()
				msg := types.NewMsgStoreCode(signer, wasmtesting.Code)

				res, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)

				expChecksums = append(expChecksums, hex.EncodeToString(res.Checksum))
			},
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			tc.malleate()

			req := &types.QueryChecksumsRequest{}
			res, err := GetSimApp(suite.chainA).WasmClientKeeper.Checksums(suite.chainA.GetContext(), req)

			if tc.expErr == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().Equal(len(expChecksums), len(res.Checksums))
				suite.Require().ElementsMatch(expChecksums, res.Checksums)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
