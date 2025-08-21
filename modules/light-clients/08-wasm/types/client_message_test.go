package types_test

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
)

func (s *TypesTestSuite) TestClientMessageValidateBasic() {
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
		s.Run(tc.name, func() {
			clientMessage := tc.clientMessage

			s.Require().Equal(types.Wasm, clientMessage.ClientType())
			err := clientMessage.ValidateBasic()

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
