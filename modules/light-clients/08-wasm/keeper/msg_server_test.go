package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

func (suite *KeeperTestSuite) TestMsgStoreCode() {
	var (
		msg    *types.MsgStoreCode
		signer string
		data   []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {
				msg = types.NewMsgStoreCode(signer, data)
			},
			nil,
		},
		{
			"fails with duplicate wasm code",
			func() {
				msg = types.NewMsgStoreCode(signer, data)

				_, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(suite.chainA.GetContext(), msg)
				suite.Require().NoError(err)
			},
			types.ErrWasmCodeExists,
		},
		{
			"fails with invalid wasm code",
			func() {
				msg = types.NewMsgStoreCode(signer, []byte{})
			},
			types.ErrWasmEmptyCode,
		},
		{
			"fails with unauthorized signer",
			func() {
				signer = suite.chainA.SenderAccount.GetAddress().String()
				msg = types.NewMsgStoreCode(signer, data)
			},
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			signer = authtypes.NewModuleAddress(govtypes.ModuleName).String()
			data, _ = os.ReadFile("../test_data/ics10_grandpa_cw.wasm.gz")

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := GetSimApp(suite.chainA).WasmClientKeeper.StoreCode(ctx, msg)
			events := ctx.EventManager().Events()

			if tc.expError == nil {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)
				suite.Require().NotEmpty(res.Checksum)

				// Verify events
				expectedEvents := sdk.Events{
					sdk.NewEvent(
						"store_wasm_code",
						sdk.NewAttribute(types.AttributeKeyWasmCodeHash, hex.EncodeToString(res.Checksum)),
					),
				}

				for _, evt := range expectedEvents {
					suite.Require().Contains(events, evt)
				}
			} else {
				suite.Require().ErrorIs(err, tc.expError)
				suite.Require().Nil(res)
				suite.Require().Empty(events)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMsgRemoveCodeHash() {
	codeHash := sha256.Sum256(wasmtesting.Code)

	govAcc := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	var msg *types.MsgRemoveCodeHash

	testCases := []struct {
		name          string
		malleate      func()
		expCodeHashes []types.CodeHash
		expFound      bool
	}{
		{
			"success",
			func() {
				msg = types.NewMsgRemoveCodeHash(govAcc, codeHash[:])
			},
			[]types.CodeHash{},
			true,
		},
		{
			"failure: code hash is missing",
			func() {
				msg = types.NewMsgRemoveCodeHash(govAcc, []byte{1})
			},
			[]types.CodeHash{codeHash[:]},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM()

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			tc.malleate()

			ctx := suite.chainA.GetContext()
			res, err := GetSimApp(suite.chainA).WasmClientKeeper.RemoveCodeHash(ctx, msg)
			events := ctx.EventManager().Events().ToABCIEvents()

			suite.Require().NoError(err)
			suite.Require().NotNil(res)
			suite.Require().Equal(tc.expFound, res.Found)

			codeHashes, err := types.GetAllCodeHashes(suite.chainA.GetContext())
			suite.Require().NoError(err)
			suite.Require().Equal(tc.expCodeHashes, codeHashes)

			// Verify events
			suite.Require().Len(events, 0)
		})
	}
}
