package types_test

import (
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

func (suite *TypesTestSuite) TestHeaderValidateBasic() {
	testCases := []struct {
		name    string
		header  *types.Header
		expPass bool
	}{
		{
			"valid header",
			&types.Header{
				Data:   []byte("data"),
				Height: clienttypes.ZeroHeight(),
			},
			true,
		},
		{
			"data is nil",
			&types.Header{
				Data:   nil,
				Height: clienttypes.ZeroHeight(),
			},
			false,
		},
		{
			"data is empty",
			&types.Header{
				Data:   []byte{},
				Height: clienttypes.ZeroHeight(),
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			header := tc.header

			suite.Require().Equal(exported.Wasm, header.ClientType())
			err := header.ValidateBasic()

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
