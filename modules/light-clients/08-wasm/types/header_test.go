package types_test

import (
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

func (suite *WasmTestSuite) TestHeaderValidateBasic() {
	testCases := []struct {
		name    string
		header  *types.Header
		expPass bool
	}{
		{
			"valid header",
			&types.Header{
				Data:   []byte("data"),
				Height: clienttypes.NewHeight(0, 0),
			},
			true,
		},
		{
			"data is nil",
			&types.Header{
				Data:   nil,
				Height: clienttypes.NewHeight(0, 0),
			},
			false,
		},
		{
			"data is empty",
			&types.Header{
				Data:   []byte(""),
				Height: clienttypes.NewHeight(0, 0),
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		header := tc.header

		suite.Require().Equal(exported.Wasm, header.ClientType())
		err := header.ValidateBasic()

		if tc.expPass {
			suite.Require().NoError(err)
		} else {
			suite.Require().Error(err)
		}
	}
}
