package types_test

import (
	errorsmod "cosmossdk.io/errors"

	wasmtesting "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
)

func (suite *TypesTestSuite) TestValidate() {
	testCases := []struct {
		name        string
		clientState *types.ClientState
		expErr      error
	}{
		{
			name:        "valid client",
			clientState: types.NewClientState([]byte{0}, wasmtesting.Code, clienttypes.ZeroHeight()),
			expErr:      nil,
		},
		{
			name:        "nil data",
			clientState: types.NewClientState(nil, wasmtesting.Code, clienttypes.ZeroHeight()),
			expErr:      errorsmod.Wrap(types.ErrInvalidData, "data cannot be empty"),
		},
		{
			name:        "empty data",
			clientState: types.NewClientState([]byte{}, wasmtesting.Code, clienttypes.ZeroHeight()),
			expErr:      errorsmod.Wrap(types.ErrInvalidData, "data cannot be empty"),
		},
		{
			name:        "nil checksum",
			clientState: types.NewClientState([]byte{0}, nil, clienttypes.ZeroHeight()),
			expErr:      errorsmod.Wrap(types.ErrInvalidChecksum, "checksum cannot be empty"),
		},
		{
			name:        "empty checksum",
			clientState: types.NewClientState([]byte{0}, []byte{}, clienttypes.ZeroHeight()),
			expErr:      errorsmod.Wrap(types.ErrInvalidChecksum, "checksum cannot be empty"),
		},
		{
			name: "longer than 32 bytes checksum",
			clientState: types.NewClientState(
				[]byte{0},
				[]byte{
					0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
					10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
					20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
					30, 31, 32, 33,
				},
				clienttypes.ZeroHeight(),
			),
			expErr: errorsmod.Wrap(types.ErrInvalidChecksum, "checksum cannot be empty"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.clientState.Validate()
			if tc.expErr == nil {
				suite.Require().NoError(err, tc.name)
			} else {
				suite.Require().Error(err, tc.name)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
