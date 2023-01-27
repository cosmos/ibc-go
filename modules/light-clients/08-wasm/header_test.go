package wasm_test

import (
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	wasm "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm"
)

func (suite *WasmTestSuite) TestHeaderValidateBasic() {
	testCases := []struct {
		name    string
		header  *wasm.Header
		expPass bool
	}{
		{
			"valid header",
			&wasm.Header{
				Data:   []byte("data"),
				Height: clienttypes.NewHeight(0, 0),
			},
			true,
		},
		{
			"data is empty",
			&wasm.Header{
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
