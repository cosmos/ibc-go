package types_test

import (
	"errors"

	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

// TestMsgProvideCounterpartyValidateBasic tests ValidateBasic for MsgProvideCounterparty
func (s *TypesTestSuite) TestMsgProvideCounterpartyValidateBasic() {
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
			"failure: invalid channel ID",
			func() {
				msg.ChannelId = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: invalid counterparty channel ID",
			func() {
				msg.CounterpartyChannelId = ""
			},
			host.ErrInvalidID,
		},
	}

	for _, tc := range testCases {
		msg = types.NewMsgProvideCounterparty(
			ibctesting.FirstClientID,
			ibctesting.SecondClientID,
			ibctesting.TestAccAddress,
		)

		tc.malleate()

		err := msg.ValidateBasic()
		expPass := tc.expError == nil
		if expPass {
			s.Require().NoError(err, "valid case %s failed", tc.name)
		} else {
			s.Require().ErrorIs(err, tc.expError, "invalid case %s passed", tc.name)
		}
	}
}

// TestMsgCreateChannelValidateBasic tests ValidateBasic for MsgCreateChannel
func (s *TypesTestSuite) TestMsgCreateChannelValidateBasic() {
	var msg *types.MsgCreateChannel

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
				msg.ClientId = ""
			},
			host.ErrInvalidID,
		},
		{
			"failure: empty key path",
			func() {
				msg.MerklePathPrefix.KeyPath = nil
			},
			errors.New("path cannot have length 0"),
		},
	}

	for _, tc := range testCases {
		msg = types.NewMsgCreateChannel(
			ibctesting.FirstClientID,
			commitmenttypes.NewMerklePath([]byte("key")),
			ibctesting.TestAccAddress,
		)

		tc.malleate()

		err := msg.ValidateBasic()
		expPass := tc.expError == nil
		if expPass {
			s.Require().NoError(err, "valid case %s failed", tc.name)
		} else {
			s.Require().ErrorContains(err, tc.expError.Error(), "invalid case %s passed", tc.name)
		}
	}
}
