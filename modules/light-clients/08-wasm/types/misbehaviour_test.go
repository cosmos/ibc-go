package types_test

import (
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

func (suite *WasmTestSuite) TestMisbehaviourValidateBasic() {
	testCases := []struct {
		name         string
		misbehaviour *wasmtypes.Misbehaviour
		expPass      bool
	}{
		{
			"valid misbehaviour",
			&wasmtypes.Misbehaviour{
				Data: []byte{0},
			},
			true,
		},
		{
			"data is nil",
			&wasmtypes.Misbehaviour{
				Data: nil,
			},
			false,
		},
		{
			"data is empty",
			&wasmtypes.Misbehaviour{
				Data: []byte{},
			},
			false,
		},
	}
	for i, tc := range testCases {
		tc := tc

		suite.Require().Equal(exported.Wasm, tc.misbehaviour.ClientType())

		if tc.expPass {
			suite.Require().NoError(tc.misbehaviour.ValidateBasic(), "valid test case %d failed: %s", i, tc.name)
		} else {
			suite.Require().Error(tc.misbehaviour.ValidateBasic(), "invalid test case %d passed: %s", i, tc.name)
		}
	}
}
