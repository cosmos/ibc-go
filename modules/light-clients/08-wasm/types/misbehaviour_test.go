package types_test

import (
	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

func (suite *TypesTestSuite) TestMisbehaviourValidateBasic() {
	testCases := []struct {
		name         string
		misbehaviour *types.Misbehaviour
		expPass      bool
	}{
		{
			"valid misbehaviour",
			&types.Misbehaviour{
				Data: []byte{0},
			},
			true,
		},
		{
			"data is nil",
			&types.Misbehaviour{
				Data: nil,
			},
			false,
		},
		{
			"data is empty",
			&types.Misbehaviour{
				Data: []byte{},
			},
			false,
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.Require().Equal(exported.Wasm, tc.misbehaviour.ClientType())

			if tc.expPass {
				suite.Require().NoError(tc.misbehaviour.ValidateBasic())
			} else {
				suite.Require().Error(tc.misbehaviour.ValidateBasic())
			}
		})
	}
}
