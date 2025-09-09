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
			name: "success: valid add msg with channel id",
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
			name: "success: valid add msg with client id",
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
			name: "success: invalid authority",
			msg: &types.MsgAddRateLimit{
				Signer:            "invalid",
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     24,
			},
			expPass: true, // Note: validate basic only checks the signer is not empty, not if it's a valid authority
		},
		{
			name: "success: empty authority",
			msg: &types.MsgAddRateLimit{
				Signer:            "",
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     24,
			},
			expPass: false,
		},
		{
			name: "failure: denom can't be empty",
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
			name: "failure: invalid client ID",
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
			name: "failure: invalid channel ID",
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
			name: "failure: max percent send > 100",
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
			name: "failure: max percent recv > 100",
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
			name: "failure: send and recv both zero",
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
			name: "failure: duration is zero hours",
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

func (s *MsgsTestSuite) TestMsgUpdateRateLimit() {
	testCases := []struct {
		name    string
		msg     *types.MsgUpdateRateLimit
		expPass bool
	}{
		{
			name: "success: valid add msg with channel id",
			msg: &types.MsgUpdateRateLimit{
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
			name: "success: valid add msg with client id",
			msg: &types.MsgUpdateRateLimit{
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
			name: "success: invalid authority",
			msg: &types.MsgUpdateRateLimit{
				Signer:            "invalid",
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     24,
			},
			expPass: true, // Note: validate basic only checks the signer is not empty, not if it's a valid authority
		},
		{
			name: "success: empty authority",
			msg: &types.MsgUpdateRateLimit{
				Signer:            "",
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
				MaxPercentSend:    sdkmath.NewInt(10),
				MaxPercentRecv:    sdkmath.NewInt(10),
				DurationHours:     24,
			},
			expPass: false,
		},
		{
			name: "failure: denom can't be empty",
			msg: &types.MsgUpdateRateLimit{
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
			name: "failure: invalid client ID",
			msg: &types.MsgUpdateRateLimit{
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
			name: "failure: invalid channel ID",
			msg: &types.MsgUpdateRateLimit{
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
			name: "failure: max percent send > 100",
			msg: &types.MsgUpdateRateLimit{
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
			name: "failure: max percent recv > 100",
			msg: &types.MsgUpdateRateLimit{
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
			name: "failure: send and recv both zero",
			msg: &types.MsgUpdateRateLimit{
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
			name: "failure: duration is zero hours",
			msg: &types.MsgUpdateRateLimit{
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

func (s *MsgsTestSuite) TestMsgRemoveRateLimit() {
	testCases := []struct {
		name    string
		msg     *types.MsgRemoveRateLimit
		expPass bool
	}{
		{
			name: "success: valid add msg with channel id",
			msg: &types.MsgRemoveRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
			},
			expPass: true,
		},
		{
			name: "success: valid add msg with client id",
			msg: &types.MsgRemoveRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: s.validClientID,
			},
			expPass: true,
		},
		{
			name: "success: invalid authority",
			msg: &types.MsgRemoveRateLimit{
				Signer:            "invalid",
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
			},
			expPass: true, // Note: validate basic only checks the signer is not empty, not if it's a valid authority
		},
		{
			name: "success: empty authority",
			msg: &types.MsgRemoveRateLimit{
				Signer:            "",
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
			},
			expPass: false,
		},
		{
			name: "failure: denom can't be empty",
			msg: &types.MsgRemoveRateLimit{
				Signer:            s.authority,
				Denom:             "",
				ChannelOrClientId: s.validChannelID,
			},
			expPass: false,
		},
		{
			name: "failure: invalid client ID",
			msg: &types.MsgRemoveRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: "invalid-client-id",
			},
			expPass: false,
		},
		{
			name: "failure: invalid channel ID",
			msg: &types.MsgRemoveRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: "channel",
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

func (s *MsgsTestSuite) TestMsgResetRateLimit() {
	testCases := []struct {
		name    string
		msg     *types.MsgResetRateLimit
		expPass bool
	}{
		{
			name: "success: valid add msg with channel id",
			msg: &types.MsgResetRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
			},
			expPass: true,
		},
		{
			name: "success: valid add msg with client id",
			msg: &types.MsgResetRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: s.validClientID,
			},
			expPass: true,
		},
		{
			name: "success: invalid authority",
			msg: &types.MsgResetRateLimit{
				Signer:            "invalid",
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
			},
			expPass: true, // Note: validate basic only checks the signer is not empty, not if it's a valid authority
		},
		{
			name: "success: empty authority",
			msg: &types.MsgResetRateLimit{
				Signer:            "",
				Denom:             "uatom",
				ChannelOrClientId: s.validChannelID,
			},
			expPass: false,
		},
		{
			name: "failure: denom can't be empty",
			msg: &types.MsgResetRateLimit{
				Signer:            s.authority,
				Denom:             "",
				ChannelOrClientId: s.validChannelID,
			},
			expPass: false,
		},
		{
			name: "failure: invalid client ID",
			msg: &types.MsgResetRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: "invalid-client-id",
			},
			expPass: false,
		},
		{
			name: "failure: invalid channel ID",
			msg: &types.MsgResetRateLimit{
				Signer:            s.authority,
				Denom:             "uatom",
				ChannelOrClientId: "channel",
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
