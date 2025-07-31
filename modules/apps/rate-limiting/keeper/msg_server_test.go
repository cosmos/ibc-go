package keeper_test

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

var (
	authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()

	addRateLimitMsg = types.MsgAddRateLimit{
		Signer:            authority,
		Denom:             "denom",
		ChannelOrClientId: "channel-0",
		MaxPercentRecv:    sdkmath.NewInt(10),
		MaxPercentSend:    sdkmath.NewInt(20),
		DurationHours:     30,
	}

	updateRateLimitMsg = types.MsgUpdateRateLimit{
		Signer:            authority,
		Denom:             "denom",
		ChannelOrClientId: "channel-0",
		MaxPercentRecv:    sdkmath.NewInt(20),
		MaxPercentSend:    sdkmath.NewInt(30),
		DurationHours:     40,
	}

	removeRateLimitMsg = types.MsgRemoveRateLimit{
		Signer:            authority,
		Denom:             "denom",
		ChannelOrClientId: "channel-0",
	}

	resetRateLimitMsg = types.MsgResetRateLimit{
		Signer:            authority,
		Denom:             "denom",
		ChannelOrClientId: "channel-0",
	}
)

// Helper function to create a channel and prevent a channel not exists error
func (s *KeeperTestSuite) createChannel(channelID string) {
	s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), transfertypes.PortID, channelID, channeltypes.Channel{})
}

// Helper function to mint tokens and create channel value to prevent a zero channel value error
func (s *KeeperTestSuite) createChannelValue(_ string, channelValue sdkmath.Int) {
	err := s.chainA.GetSimApp().BankKeeper.MintCoins(s.chainA.GetContext(), minttypes.ModuleName, sdk.NewCoins(sdk.NewCoin(addRateLimitMsg.Denom, channelValue)))
	s.Require().NoError(err)
}

// Helper function to add a rate limit with an optional error expectation
func (s *KeeperTestSuite) addRateLimit(msgAddRateLimit types.MsgAddRateLimit, expectedErr *errorsmod.Error) {
	msgServer := keeper.NewMsgServerImpl(s.chainA.GetSimApp().RateLimitKeeper)
	_, actualErr := msgServer.AddRateLimit(s.chainA.GetContext(), &msgAddRateLimit)

	// If it should have been added successfully, confirm no error
	// and confirm the rate limit was created
	if expectedErr == nil {
		s.Require().NoError(actualErr)

		_, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), addRateLimitMsg.Denom, addRateLimitMsg.ChannelOrClientId)
		s.Require().True(found)
	} else {
		// If it should have failed, check the error
		s.Require().ErrorIs(actualErr, expectedErr)
	}
}

// Helper function to add a rate limit successfully
func (s *KeeperTestSuite) addRateLimitSuccessful(msgAddRateLimit types.MsgAddRateLimit) {
	s.addRateLimit(msgAddRateLimit, nil)
}

// Helper function to add a rate limit with an expected error
func (s *KeeperTestSuite) addRateLimitWithError(msgAddRateLimit types.MsgAddRateLimit, expectedErr *errorsmod.Error) {
	s.addRateLimit(msgAddRateLimit, expectedErr)
}

func (s *KeeperTestSuite) TestMsgServer_AddRateLimit() {
	denom := addRateLimitMsg.Denom
	channelID := addRateLimitMsg.ChannelOrClientId
	channelValue := sdkmath.NewInt(100)

	// First try to add a rate limit when there's no channel value, it will fail
	s.addRateLimitWithError(addRateLimitMsg, types.ErrZeroChannelValue)

	// Create channel value
	s.createChannelValue(denom, channelValue)

	// Then try to add a rate limit before the channel has been created, it will also fail
	s.addRateLimitWithError(addRateLimitMsg, types.ErrChannelNotFound)

	// Create the channel
	s.createChannel(channelID)

	// Now add a rate limit successfully
	s.addRateLimitSuccessful(addRateLimitMsg)

	// Finally, try to add the same rate limit again - it should fail
	s.addRateLimitWithError(addRateLimitMsg, types.ErrRateLimitAlreadyExists)

	// Verify that signer == authority required
	invalidSignerMsg := addRateLimitMsg
	invalidSignerMsg.Signer = ""
	s.addRateLimitWithError(invalidSignerMsg, govtypes.ErrInvalidSigner)
}

func (s *KeeperTestSuite) TestMsgServer_UpdateRateLimit() {
	denom := updateRateLimitMsg.Denom
	channelID := updateRateLimitMsg.ChannelOrClientId
	channelValue := sdkmath.NewInt(100)

	msgServer := keeper.NewMsgServerImpl(s.chainA.GetSimApp().RateLimitKeeper)

	// Create channel and channel value
	s.createChannel(channelID)
	s.createChannelValue(denom, channelValue)

	// Attempt to update a rate limit that does not exist
	_, err := msgServer.UpdateRateLimit(s.chainA.GetContext(), &updateRateLimitMsg)
	s.Require().Equal(err, types.ErrRateLimitNotFound)

	// Add a rate limit successfully
	s.addRateLimitSuccessful(addRateLimitMsg)

	// Update the rate limit successfully
	_, err = msgServer.UpdateRateLimit(s.chainA.GetContext(), &updateRateLimitMsg)
	s.Require().NoError(err)

	// Check ratelimit quota is updated correctly
	updatedRateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denom, channelID)
	s.Require().True(found)
	s.Require().Equal(updatedRateLimit.Quota, &types.Quota{
		MaxPercentSend: updateRateLimitMsg.MaxPercentSend,
		MaxPercentRecv: updateRateLimitMsg.MaxPercentRecv,
		DurationHours:  updateRateLimitMsg.DurationHours,
	})

	// Attempt to update a rate limit that has invalid authority
	invalidSignerMsg := updateRateLimitMsg
	invalidSignerMsg.Signer = ""
	_, err = msgServer.UpdateRateLimit(s.chainA.GetContext(), &invalidSignerMsg)
	s.Require().ErrorIs(err, govtypes.ErrInvalidSigner)
}

func (s *KeeperTestSuite) TestMsgServer_RemoveRateLimit() {
	denom := removeRateLimitMsg.Denom
	channelID := removeRateLimitMsg.ChannelOrClientId
	channelValue := sdkmath.NewInt(100)

	msgServer := keeper.NewMsgServerImpl(s.chainA.GetSimApp().RateLimitKeeper)

	s.createChannel(channelID)
	s.createChannelValue(denom, channelValue)

	// Attempt to remove a rate limit that does not exist
	_, err := msgServer.RemoveRateLimit(s.chainA.GetContext(), &removeRateLimitMsg)
	s.Require().Equal(err, types.ErrRateLimitNotFound)

	// Add a rate limit successfully
	s.addRateLimitSuccessful(addRateLimitMsg)

	// Remove the rate limit successfully
	_, err = msgServer.RemoveRateLimit(s.chainA.GetContext(), &removeRateLimitMsg)
	s.Require().NoError(err)

	// Confirm it was removed
	_, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denom, channelID)
	s.Require().False(found)

	// Attempt to Remove a rate limit that has invalid authority
	invalidSignerMsg := removeRateLimitMsg
	invalidSignerMsg.Signer = ""
	_, err = msgServer.RemoveRateLimit(s.chainA.GetContext(), &invalidSignerMsg)
	s.Require().ErrorIs(err, govtypes.ErrInvalidSigner)
}

func (s *KeeperTestSuite) TestMsgServer_ResetRateLimit() {
	denom := resetRateLimitMsg.Denom
	channelID := resetRateLimitMsg.ChannelOrClientId
	channelValue := sdkmath.NewInt(100)

	msgServer := keeper.NewMsgServerImpl(s.chainA.GetSimApp().RateLimitKeeper)

	s.createChannel(channelID)
	s.createChannelValue(denom, channelValue)

	// Attempt to reset a rate limit that does not exist
	_, err := msgServer.ResetRateLimit(s.chainA.GetContext(), &resetRateLimitMsg)
	s.Require().Equal(err, types.ErrRateLimitNotFound)

	// Add a rate limit successfully
	s.addRateLimitSuccessful(addRateLimitMsg)

	// Reset the rate limit successfully
	_, err = msgServer.ResetRateLimit(s.chainA.GetContext(), &resetRateLimitMsg)
	s.Require().NoError(err)

	// Check ratelimit quota is flow correctly
	resetRateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denom, channelID)
	s.Require().True(found)
	s.Require().Equal(resetRateLimit.Flow, &types.Flow{
		Inflow:       sdkmath.ZeroInt(),
		Outflow:      sdkmath.ZeroInt(),
		ChannelValue: channelValue,
	})

	// Attempt to Remove a rate limit that has invalid authority
	invalidSignerMsg := resetRateLimitMsg
	invalidSignerMsg.Signer = ""
	_, err = msgServer.ResetRateLimit(s.chainA.GetContext(), &invalidSignerMsg)
	s.Require().ErrorIs(err, govtypes.ErrInvalidSigner)
}
