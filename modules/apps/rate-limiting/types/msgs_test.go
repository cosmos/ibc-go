package types_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

type MsgsTestSuite struct {
	suite.Suite

	authority      string
	randomAddress  string
	validChannelID string
	validClientID  string
}

func (s *MsgsTestSuite) SetupTest() {
	s.authority = "cosmos10h9stc5v6ntgeygf5xf945njqq5h32r53uquvw"
	s.randomAddress = "cosmos10h9stc5v6ntgeygf5xf945njqq5h32r53uquvw"
	s.validChannelID = "channel-0"
	s.validClientID = "07-tendermint-0"
}

func TestMsgsTestSuite(t *testing.T) {
	suite.Run(t, new(MsgsTestSuite))
}

func (s *MsgsTestSuite) TestMsgAddRateLimit() {
	testCases := []struct {
		name    string
		msg     *types.MsgAddRateLimit
		expPass bool
	}{
		{
			name: "valid add msg with channel id",
			msg: &types.MsgAddRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     24,
			},
			expPass: true,
		},
		{
			name: "valid add msg with client id",
			msg: &types.MsgAddRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: s.validClientID,
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     24,
			},
			expPass: true,
		},
		{
			name: "invalid authority",
			msg: &types.MsgAddRateLimit{
				Signer:            "invalid",
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     24,
			},
			expPass: false,
		},
		{
			name: "denom can't be empty",
			msg: &types.MsgAddRateLimit{
				Signer:            s.authority,
				Denom:             "",
				ChannelOrClientId: s.validChannelID,
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     24,
			},
			expPass: false,
		},
		{
			name: "invalid client ID",
			msg: &types.MsgAddRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: "invalid-client-id",
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     24,
			},
			expPass: false,
		},
		{
			name: "invalid channel ID",
			msg: &types.MsgAddRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: "channel",
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     24,
			},
			expPass: false,
		},
		{
			name: "max percent send > 100",
			msg: &types.MsgAddRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
				MaxPercentSend:    sdkmath.NewInt(101),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     24,
			},
			expPass: false,
		},
		{
			name: "max percent recv > 100",
			msg: &types.MsgAddRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(101),
				DurationHours:     24,
			},
			expPass: false,
		},
		{
			name: "send and recv both zero",
			msg: &types.MsgAddRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
				MaxPercentSend:    sdkmath.ZeroInt(),
				MaxPercentRecv:    sdkmath.ZeroInt(),
				DurationHours:     24,
			},
			expPass: false,
		},
		{
			name: "duration is zero hours",
			msg: &types.MsgAddRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     0,
			},
			expPass: false,
		},
	}

	for i, tc := range testCases {
		err := tc.msg.ValidateBasic()
		if tc.expPass {
			s.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
		} else {
			s.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

func (s *MsgsTestSuite) TestMsgUpdateParams() {
	validParams := types.Params{
		Enabled:           true,
		DefaultMaxOutflow: "1000000",
		DefaultMaxInflow:  "1000000",
		DefaultPeriod:     86400,
	}

	invalidParams := types.Params{
		Enabled:           true,
		DefaultMaxOutflow: "",
		DefaultMaxInflow:  "1000000",
		DefaultPeriod:     86400,
	}

	testCases := []struct {
		name    string
		msg     *types.MsgUpdateParams
		expPass bool
	}{
		{
			name: "valid update params msg",
			msg: &types.MsgUpdateParams{
				Signer: s.authority,
				Params: validParams,
			},
			expPass: true,
		},
		{
			name: "invalid signer",
			msg: &types.MsgUpdateParams{
				Signer: "invalid",
				Params: validParams,
			},
			expPass: false,
		},
		{
			name: "invalid params",
			msg: &types.MsgUpdateParams{
				Signer: s.authority,
				Params: invalidParams,
			},
			expPass: false,
		},
	}

	for i, tc := range testCases {
		err := tc.msg.ValidateBasic()
		if tc.expPass {
			s.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
		} else {
			s.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
		}

		// Test GetSigners
		if tc.expPass {
			signers := tc.msg.GetSigners()
			s.Require().Len(signers, 1)
			s.Require().Equal(s.authority, signers[0].String())
		}

		// Test ProtoMessage
		s.Require().NotPanics(func() {
			tc.msg.ProtoMessage()
		})

		// Test Reset
		s.Require().NotPanics(func() {
			emptyMsg := &types.MsgUpdateParams{}
			tc.msg.Reset()
			s.Require().Equal(emptyMsg, tc.msg)
		})

		// Test String
		s.Require().NotPanics(func() {
			_ = tc.msg.String()
		})
	}
}
