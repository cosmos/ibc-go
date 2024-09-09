package types_test

import (

	//nolint:staticcheck

	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

// TestMsgProvideCounterpartyValidateBasic tests ValidateBasic for MsgProvideCounterparty
func (suite *TypesTestSuite) TestMsgProvideCounterpartyValidateBasic() {
	var msg *types.MsgProvideCounterparty

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"failure: invalid signer address",
			func() {
				msg.Signer = "invalid"
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: invalid client ID",
			func() {
				msg.ChannelId = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: invalid counterparty client ID",
			func() {
				msg.Counterparty.ClientId = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: empty key path of counterparty of merkle path prefix",
			func() {
				msg.Counterparty.MerklePathPrefix.KeyPath = nil
			},
			types.ErrInvalidCounterparty,
		},
	}

	for _, tc := range testCases {
		msg = types.NewMsgProvideCounterparty(
			ibctesting.TestAccAddress,
			ibctesting.FirstClientID,
			ibctesting.SecondClientID,
			commitmenttypes.NewMerklePath([]byte("key")),
		)

		tc.malleate()

		err := msg.ValidateBasic()
		expPass := tc.expError == nil
		if expPass {
			suite.Require().NoError(err, "valid case %s failed", tc.name)
		} else {
			suite.Require().ErrorIs(err, tc.expError, "invalid case %s passed", tc.name)
		}
	}
}
