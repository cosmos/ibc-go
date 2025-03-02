package types_test

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
)

func (suite *TypesTestSuite) TestClientMessageValidateBasic() {
	testCases := []struct {
		name          string
		clientMessage *types.ClientMessage
		expErr        error
	}{
		{
			"valid client message",
			&types.ClientMessage{
				Data: []byte("data"),
			},
			nil,
		},
		{
			"data is nil",
			&types.ClientMessage{
				Data: nil,
			},
			errorsmod.Wrap(types.ErrInvalidData, "data cannot be empty"),
		},
		{
			"data is empty",
			&types.ClientMessage{
				Data: []byte{},
			},
			errorsmod.Wrap(types.ErrInvalidData, "data cannot be empty"),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			clientMessage := tc.clientMessage

			suite.Require().Equal(types.Wasm, clientMessage.ClientType())
			err := clientMessage.ValidateBasic()

			if tc.expErr == nil {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
