package keeper_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	// "time"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"

	// sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	// clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	// ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const (
	transferPort    = "transfer"
	uosmo           = "uosmo"
	ujuno           = "ujuno"
	ustrd           = "ustrd"
	stuatom         = "stuatom"
	channelOnStride = "channel-0"
	channelOnHost   = "channel-1"
)

func hashDenomTrace(denomTrace string) string {
	trace32byte := sha256.Sum256([]byte(denomTrace))
	var traceTmByte tmbytes.HexBytes = trace32byte[:]
	return fmt.Sprintf("ibc/%s", traceTmByte)
}

func TestParseDenomFromSendPacket(t *testing.T) {
	testCases := []struct {
		name             string
		packetDenomTrace string
		expectedDenom    string
	}{
		// Native assets stay as is
		{
			name:             "ustrd",
			packetDenomTrace: ustrd,
			expectedDenom:    ustrd,
		},
		{
			name:             "stuatom",
			packetDenomTrace: stuatom,
			expectedDenom:    stuatom,
		},
		// Non-native assets are hashed
		{
			name:             "uosmo_one_hop",
			packetDenomTrace: "transfer/channel-0/usomo",
			expectedDenom:    hashDenomTrace("transfer/channel-0/usomo"),
		},
		{
			name:             "uosmo_two_hops",
			packetDenomTrace: "transfer/channel-2/transfer/channel-1/usomo",
			expectedDenom:    hashDenomTrace("transfer/channel-2/transfer/channel-1/usomo"),
		},
		// IBC denoms are passed through as is
		{
			name:             "ibc_denom",
			packetDenomTrace: "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2",
			expectedDenom:    "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			packet := transfertypes.FungibleTokenPacketData{
				Denom: tc.packetDenomTrace,
			}

			parsedDenom := keeper.ParseDenomFromSendPacket(packet)
			require.Equal(t, tc.expectedDenom, parsedDenom, tc.name)
		})
	}
}

func TestParseDenomFromRecvPacket(t *testing.T) {
	osmoChannelOnStride := "channel-0"
	strideChannelOnOsmo := "channel-100"
	junoChannelOnOsmo := "channel-200"
	junoChannelOnStride := "channel-300"

	testCases := []struct {
		name               string
		packetDenomTrace   string
		sourceChannel      string
		destinationChannel string
		expectedDenom      string
	}{
		// Sink asset one hop away:
		//   uosmo sent from Osmosis to Stride (uosmo)
		//   -> tack on prefix (transfer/channel-0/uosmo) and hash
		{
			name:               "sink_one_hop",
			packetDenomTrace:   uosmo,
			sourceChannel:      strideChannelOnOsmo,
			destinationChannel: osmoChannelOnStride,
			expectedDenom:      hashDenomTrace(fmt.Sprintf("%s/%s/%s", transferPort, osmoChannelOnStride, uosmo)),
		},
		// Sink asset two hops away:
		//   ujuno sent from Juno to Osmosis to Stride (transfer/channel-200/ujuno)
		//   -> tack on prefix (transfer/channel-0/transfer/channel-200/ujuno) and hash
		{
			name:               "sink_two_hops",
			packetDenomTrace:   fmt.Sprintf("%s/%s/%s", transferPort, junoChannelOnOsmo, ujuno),
			sourceChannel:      strideChannelOnOsmo,
			destinationChannel: osmoChannelOnStride,
			expectedDenom:      hashDenomTrace(fmt.Sprintf("%s/%s/%s/%s/%s", transferPort, osmoChannelOnStride, transferPort, junoChannelOnOsmo, ujuno)),
		},
		// Native source assets
		//    ustrd sent from Stride to Osmosis and then back to Stride (transfer/channel-0/ustrd)
		//    -> remove prefix and leave as is (ustrd)
		{
			name:               "native_source",
			packetDenomTrace:   fmt.Sprintf("%s/%s/%s", transferPort, strideChannelOnOsmo, ustrd),
			sourceChannel:      strideChannelOnOsmo,
			destinationChannel: osmoChannelOnStride,
			expectedDenom:      ustrd,
		},
		// Non-native source assets
		//    ujuno was sent from Juno to Stride, then to Osmosis, then back to Stride (transfer/channel-0/transfer/channel-300/ujuno)
		//    -> remove prefix (transfer/channel-300/ujuno) and hash
		{
			name:               "non_native_source",
			packetDenomTrace:   fmt.Sprintf("%s/%s/%s/%s/%s", transferPort, strideChannelOnOsmo, transferPort, junoChannelOnStride, ujuno),
			sourceChannel:      strideChannelOnOsmo,
			destinationChannel: osmoChannelOnStride,
			expectedDenom:      hashDenomTrace(fmt.Sprintf("%s/%s/%s", transferPort, junoChannelOnStride, ujuno)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			packet := channeltypes.Packet{
				SourcePort:         transferPort,
				DestinationPort:    transferPort,
				SourceChannel:      tc.sourceChannel,
				DestinationChannel: tc.destinationChannel,
			}
			packetData := transfertypes.FungibleTokenPacketData{
				Denom: tc.packetDenomTrace,
			}

			parsedDenom := keeper.ParseDenomFromRecvPacket(packet, packetData)
			require.Equal(t, tc.expectedDenom, parsedDenom, tc.name)
		})
	}
}

func (s *KeeperTestSuite) TestParsePacketInfo() {
	sourceChannel := "channel-100"
	destinationChannel := "channel-200"
	denom := "denom"
	amountString := "100"
	amountInt := sdkmath.NewInt(100)
	sender := "sender"
	receiver := "receiver"

	packetData, err := json.Marshal(transfertypes.FungibleTokenPacketData{
		Denom:    denom,
		Amount:   amountString,
		Sender:   sender,
		Receiver: receiver,
	})
	s.Require().NoError(err)

	packet := channeltypes.Packet{
		SourcePort:         transferPort,
		SourceChannel:      sourceChannel,
		DestinationPort:    transferPort,
		DestinationChannel: destinationChannel,
		Data:               packetData,
	}

	// Send 'denom' from channel-100 (stride) -> channel-200
	// Since the 'denom' is native, it's kept as is for the rate limit object
	expectedSendPacketInfo := keeper.RateLimitedPacketInfo{
		ChannelID: sourceChannel,
		Denom:     denom,
		Amount:    amountInt,
		Sender:    sender,
		Receiver:  receiver,
	}
	actualSendPacketInfo, err := keeper.ParsePacketInfo(packet, types.PACKET_SEND)
	s.Require().NoError(err, "no error expected when parsing send packet")
	s.Require().Equal(expectedSendPacketInfo, actualSendPacketInfo, "send packet")

	// Receive 'denom' from channel-100 -> channel-200 (stride)
	// The stride channel (channel-200) should be tacked onto the end and the denom should be hashed
	expectedRecvPacketInfo := keeper.RateLimitedPacketInfo{
		ChannelID: destinationChannel,
		Denom:     hashDenomTrace(fmt.Sprintf("transfer/%s/%s", destinationChannel, denom)),
		Amount:    amountInt,
		Sender:    sender,
		Receiver:  receiver,
	}
	actualRecvPacketInfo, err := keeper.ParsePacketInfo(packet, types.PACKET_RECV)
	s.Require().NoError(err, "no error expected when parsing recv packet")
	s.Require().Equal(expectedRecvPacketInfo, actualRecvPacketInfo, "recv packet")
}

func (s *KeeperTestSuite) TestCheckAcknowledementSucceeded() {
	// Test case 1: Successful ack (legacy format)
	ackSuccessLegacy := transfertypes.ModuleCdc.MustMarshalJSON(&channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Result{Result: []byte{1}},
	})
	success, err := s.chainA.GetSimApp().RateLimitKeeper.CheckAcknowledementSucceeded(s.chainA.GetContext(), ackSuccessLegacy)
	s.Require().NoError(err, "no error expected for successful legacy ack")
	s.Require().True(success, "successful legacy ack should return true")

	// Test case 2: Failed ack (legacy format)
	ackFailLegacy := transfertypes.ModuleCdc.MustMarshalJSON(&channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{Error: "some error"},
	})
	success, err = s.chainA.GetSimApp().RateLimitKeeper.CheckAcknowledementSucceeded(s.chainA.GetContext(), ackFailLegacy)
	s.Require().NoError(err, "no error expected for failed legacy ack")
	s.Require().False(success, "failed legacy ack should return false")

	// Test case 3: Failed ack (IBC v2 universal format)
	ackFailV2 := channeltypesv2.ErrorAcknowledgement[:] // Use the predefined error ack bytes
	success, err = s.chainA.GetSimApp().RateLimitKeeper.CheckAcknowledementSucceeded(s.chainA.GetContext(), ackFailV2)
	s.Require().NoError(err, "no error expected for failed v2 ack")
	s.Require().False(success, "failed v2 ack should return false")

	// Test case 4: Invalid ack format
	invalidAck := []byte("invalid ack")
	success, err = s.chainA.GetSimApp().RateLimitKeeper.CheckAcknowledementSucceeded(s.chainA.GetContext(), invalidAck)
	s.Require().Error(err, "error expected for invalid ack format")
	s.Require().False(success, "invalid ack should return false")
}

func (s *KeeperTestSuite) createRateLimitCloseToQuota(denom string, channelId string, direction types.PacketDirection) {
	channelValue := sdkmath.NewInt(100)
	threshold := sdkmath.NewInt(10)

	// Set inflow/outflow close to threshold, depending on which direction we're going in
	inflow := sdkmath.ZeroInt()
	outflow := sdkmath.ZeroInt()
	if direction == types.PACKET_RECV {
		inflow = sdkmath.NewInt(9)
	} else {
		outflow = sdkmath.NewInt(9)
	}

	// Store rate limit
	s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{
		Path: &types.Path{
			Denom:             denom,
			ChannelOrClientId: channelId,
		},
		Quota: &types.Quota{
			MaxPercentSend: threshold,
			MaxPercentRecv: threshold,
		},
		Flow: &types.Flow{
			Inflow:       inflow,
			Outflow:      outflow,
			ChannelValue: channelValue,
		},
	})
}

func (s *KeeperTestSuite) TestSendRateLimitedPacket() {
	// For send packets, the source will be stride and the destination will be the host
	denom := ustrd
	sourceChannel := channelOnStride
	destinationChannel := channelOnHost
	amountToExceed := "5"
	sequence := uint64(10)

	// Create rate limit (for SEND, use SOURCE channel)
	s.createRateLimitCloseToQuota(denom, sourceChannel, types.PACKET_SEND)

	// This packet should cause an Outflow quota exceed error
	packetData, err := json.Marshal(transfertypes.FungibleTokenPacketData{Denom: denom, Amount: amountToExceed})
	s.Require().NoError(err)
	packet := channeltypes.Packet{
		SourcePort:         transferPort,
		SourceChannel:      sourceChannel,
		DestinationPort:    transferPort,
		DestinationChannel: destinationChannel,
		Data:               packetData,
		Sequence:           sequence,
	}

	// We check for a quota error because it doesn't appear until the end of the function
	// We're avoiding checking for a success here because we can get a false positive if the rate limit doesn't exist
	err = s.chainA.GetSimApp().RateLimitKeeper.SendRateLimitedPacket(s.chainA.GetContext(), packet)
	s.Require().ErrorIs(err, types.ErrQuotaExceeded, "error type")
	s.Require().ErrorContains(err, "Outflow exceeds quota", "error text")

	// Reset the rate limit and try again
	err = s.chainA.GetSimApp().RateLimitKeeper.ResetRateLimit(s.chainA.GetContext(), denom, channelId)
	s.Require().NoError(err, "no error expected when resetting rate limit")

	err = s.chainA.GetSimApp().RateLimitKeeper.SendRateLimitedPacket(s.chainA.GetContext(), packet)
	s.Require().NoError(err, "no error expected when sending packet after reset")

	// Check that the pending packet was stored
	found := s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), sourceChannel, sequence)
	s.Require().True(found, "pending send packet")
}

func (s *KeeperTestSuite) TestReceiveRateLimitedPacket() {
	// For receive packets, the source will be the host and the destination will be stride
	packetDenom := uosmo
	sourceChannel := channelOnHost
	destinationChannel := channelOnStride
	amountToExceed := "5"

	// When the packet is received, the port and channel prefix will be added and the denom will be hashed
	//  before the rate limit is found from the store
	rateLimitDenom := hashDenomTrace(fmt.Sprintf("%s/%s/%s", transferPort, channelOnStride, packetDenom))

	// Create rate limit (for RECV, use DESTINATION channel)
	s.createRateLimitCloseToQuota(rateLimitDenom, destinationChannel, types.PACKET_RECV)

	// This packet should cause an Outflow quota exceed error
	packetData, err := json.Marshal(transfertypes.FungibleTokenPacketData{Denom: packetDenom, Amount: amountToExceed})
	s.Require().NoError(err)
	packet := channeltypes.Packet{
		SourcePort:         transferPort,
		SourceChannel:      sourceChannel,
		DestinationPort:    transferPort,
		DestinationChannel: destinationChannel,
		Data:               packetData,
	}

	// We check for a quota error because it doesn't appear until the end of the function
	// We're avoiding checking for a success here because we can get a false positive if the rate limit doesn't exist
	err = s.chainA.GetSimApp().RateLimitKeeper.ReceiveRateLimitedPacket(s.chainA.GetContext(), packet)
	s.Require().ErrorIs(err, types.ErrQuotaExceeded, "error type")
	s.Require().ErrorContains(err, "Inflow exceeds quota", "error text")
}

func (s *KeeperTestSuite) TestAcknowledgeRateLimitedPacket_AckSuccess() {
	// For ack packets, the source will be stride and the destination will be the host
	denom := ustrd
	sourceChannel := channelOnStride
	destinationChannel := channelOnHost
	sequence := uint64(10)

	// Create rate limit - the flow and quota does not matter for this test
	s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{
		Path: &types.Path{Denom: denom, ChannelOrClientId: channelId},
	})

	// Store the pending packet for this sequence number
	s.chainA.GetSimApp().RateLimitKeeper.SetPendingSendPacket(s.chainA.GetContext(), sourceChannel, sequence)

	// Build the ack packet
	packetData, err := json.Marshal(transfertypes.FungibleTokenPacketData{Denom: denom, Amount: "10"})
	s.Require().NoError(err)
	packet := channeltypes.Packet{
		SourcePort:         transferPort,
		SourceChannel:      sourceChannel,
		DestinationPort:    transferPort,
		DestinationChannel: destinationChannel,
		Data:               packetData,
		Sequence:           sequence,
	}
	ackSuccess := transfertypes.ModuleCdc.MustMarshalJSON(&channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Result{Result: []byte{1}},
	})

	// Call AckPacket with the successful ack
	err = s.chainA.GetSimApp().RateLimitKeeper.AcknowledgeRateLimitedPacket(s.chainA.GetContext(), packet, ackSuccess)
	s.Require().NoError(err, "no error expected during AckPacket")

	// Confirm the pending packet was removed
	found := s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), sourceChannel, sequence)
	s.Require().False(found, "send packet should have been removed")
}

func (s *KeeperTestSuite) TestAcknowledgeRateLimitedPacket_AckFailure() {
	// For ack packets, the source will be stride and the destination will be the host
	denom := ustrd
	sourceChannel := channelOnStride
	destinationChannel := channelOnHost
	initialOutflow := sdkmath.NewInt(100)
	packetAmount := sdkmath.NewInt(10)
	sequence := uint64(10)

	// Create rate limit - only outflow is needed to this tests
	s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{
		Path: &types.Path{Denom: denom, ChannelOrClientId: channelId},
		Flow: &types.Flow{Outflow: initialOutflow},
	})

	// Store the pending packet for this sequence number
	s.chainA.GetSimApp().RateLimitKeeper.SetPendingSendPacket(s.chainA.GetContext(), sourceChannel, sequence)

	// Build the ack packet
	packetData, err := json.Marshal(transfertypes.FungibleTokenPacketData{Denom: denom, Amount: packetAmount.String()})
	s.Require().NoError(err)
	packet := channeltypes.Packet{
		SourcePort:         transferPort,
		SourceChannel:      sourceChannel,
		DestinationPort:    transferPort,
		DestinationChannel: destinationChannel,
		Data:               packetData,
		Sequence:           sequence,
	}
	ackFailure := transfertypes.ModuleCdc.MustMarshalJSON(&channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{Error: "error"},
	})

	// Call OnTimeoutPacket with the failed ack
	err = s.chainA.GetSimApp().RateLimitKeeper.AcknowledgeRateLimitedPacket(s.chainA.GetContext(), packet, ackFailure)
	s.Require().NoError(err, "no error expected during AckPacket")

	// Confirm the pending packet was removed
	found := s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), sourceChannel, sequence)
	s.Require().False(found, "send packet should have been removed")

	// Confirm the flow was adjusted
	rateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denom, sourceChannel)
	s.Require().True(found)
	s.Require().Equal(initialOutflow.Sub(packetAmount).Int64(), rateLimit.Flow.Outflow.Int64(), "outflow")
}

func (s *KeeperTestSuite) TestTimeoutRateLimitedPacket() {
	// For timeout packets, the source will be stride and the destination will be the host
	denom := ustrd
	sourceChannel := channelOnStride
	destinationChannel := channelOnHost
	initialOutflow := sdkmath.NewInt(100)
	packetAmount := sdkmath.NewInt(10)
	sequence := uint64(10)

	// Create rate limit - only outflow is needed to this tests
	s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{
		Path: &types.Path{Denom: denom, ChannelOrClientId: channelId},
		Flow: &types.Flow{Outflow: initialOutflow},
	})

	// Store the pending packet for this sequence number
	s.chainA.GetSimApp().RateLimitKeeper.SetPendingSendPacket(s.chainA.GetContext(), sourceChannel, sequence)

	// Build the timeout packet
	packetData, err := json.Marshal(transfertypes.FungibleTokenPacketData{Denom: denom, Amount: packetAmount.String()})
	s.Require().NoError(err)
	packet := channeltypes.Packet{
		SourcePort:         transferPort,
		SourceChannel:      sourceChannel,
		DestinationPort:    transferPort,
		DestinationChannel: destinationChannel,
		Data:               packetData,
		Sequence:           sequence,
	}

	// Call OnTimeoutPacket - the outflow should get decremented
	err = s.chainA.GetSimApp().RateLimitKeeper.TimeoutRateLimitedPacket(s.chainA.GetContext(), packet)
	s.Require().NoError(err, "no error expected when calling timeout packet")

	expectedOutflow := initialOutflow.Sub(packetAmount)
	rateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denom, channelId)
	s.Require().True(found)
	s.Require().Equal(expectedOutflow.Int64(), rateLimit.Flow.Outflow.Int64(), "outflow decremented")

	// Check that the pending packet has been removed
	found = s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), channelId, sequence)
	s.Require().False(found, "pending packet should have been removed")

	// Call OnTimeoutPacket again with a different sequence number
	// (to simulate a timeout that arrived in a different quota window from where the send occurred)
	// The outflow should not change
	packet.Sequence -= 1
	err = s.chainA.GetSimApp().RateLimitKeeper.TimeoutRateLimitedPacket(s.chainA.GetContext(), packet)
	s.Require().NoError(err, "no error expected when calling timeout packet again")

	rateLimit, found = s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denom, channelId)
	s.Require().True(found)
	s.Require().Equal(expectedOutflow.Int64(), rateLimit.Flow.Outflow.Int64(), "outflow should not have changed")
}

// // --- Middleware Tests ---

// // TestOnRecvPacket_Allowed tests the middleware's OnRecvPacket when the packet is allowed
// func (s *KeeperTestSuite) TestOnRecvPacket_Allowed() {
// 	path := ibctesting.NewPath(s.chainA, s.chainB) // Use ibctesting
// 	path.Setup()

// 	// Define recipient and calculate expected voucher denom on chain B
// 	recipientAddr := s.chainB.SenderAccount.GetAddress()
// 	voucherDenomStr := hashDenomTrace(fmt.Sprintf("%s/%s/%s", transferPort, path.EndpointB.ChannelID, uosmo))

// 	// Fund recipient account with native denom
// 	fundAmount := sdkmath.NewInt(1000000)
// 	bondDenom, err := s.chainB.GetSimApp().StakingKeeper.BondDenom(s.chainB.GetContext())
// 	s.Require().NoError(err, "getting bond denom failed")
// 	fundCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, fundAmount))
// 	// Mint native denom to transfer module
// 	err = s.chainB.GetSimApp().BankKeeper.MintCoins(s.chainB.GetContext(), transfertypes.ModuleName, fundCoins)
// 	s.Require().NoError(err, "minting native denom coins to transfer module failed")
// 	// Send native denom from transfer module to recipient
// 	err = s.chainB.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(s.chainB.GetContext(), transfertypes.ModuleName, recipientAddr, fundCoins)
// 	s.Require().NoError(err, "funding recipient account with native denom failed")

// 	// Create the test packet data
// 	testAmountStr := "10"
// 	testAmountInt, _ := sdkmath.NewIntFromString(testAmountStr)
// 	packetDataBz, err := json.Marshal(transfertypes.FungibleTokenPacketData{Denom: uosmo, Amount: testAmountStr, Receiver: recipientAddr.String()})
// 	s.Require().NoError(err)

// 	// Set the rate limit using the voucher denom string
// 	simulatedSupply := sdkmath.NewInt(1000) // Keep simulated supply for rate limit calculation
// 	s.chainB.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainB.GetContext(), types.RateLimit{
// 		Path:  &types.Path{Denom: voucherDenomStr, ChannelOrClientId: path.EndpointB.ChannelID},
// 		Quota: &types.Quota{MaxPercentRecv: sdkmath.NewInt(100), DurationHours: 1}, // High quota
// 		Flow:  &types.Flow{Inflow: sdkmath.ZeroInt(), Outflow: sdkmath.ZeroInt(), ChannelValue: simulatedSupply},
// 	})

// 	// Create the packet
// 	packet := channeltypes.Packet{
// 		Sequence:           1,
// 		SourcePort:         path.EndpointA.ChannelConfig.PortID,
// 		SourceChannel:      path.EndpointA.ChannelID,
// 		DestinationPort:    path.EndpointB.ChannelConfig.PortID,
// 		DestinationChannel: path.EndpointB.ChannelID,
// 		Data:               packetDataBz, // Correct variable
// 		TimeoutHeight:      clienttypes.ZeroHeight(),
// 		TimeoutTimestamp:   uint64(s.coordinator.CurrentTime.Add(time.Hour).UnixNano()),
// 	}

// 	// Relay the packet. This will call OnRecvPacket on chain B through the integrated middleware stack.
// 	err = path.RelayPacket(packet)
// 	s.Require().NoError(err, "relaying packet failed") // Check relay error first

// 	// Check acknowledgement on chain B
// 	ack, found := s.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetPacketAcknowledgement(s.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
// 	s.Require().True(found, "acknowledgement not found")
// 	s.Require().NotNil(ack, "ack should not be nil")

// 	// Check acknowledgement success
// 	var ackData channeltypes.Acknowledgement
// 	// Use transfertypes codec to unmarshal
// 	err = transfertypes.ModuleCdc.UnmarshalJSON(ack, &ackData)
// 	s.Require().NoError(err, "unmarshalling acknowledgement failed")
// 	s.Require().True(ackData.Success(), "ack should be successful")

// 	// Check flow was updated
// 	rateLimit, found := s.chainB.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainB.GetContext(), voucherDenomStr, path.EndpointB.ChannelID)
// 	s.Require().True(found)
// 	s.Require().Equal(testAmountInt.Int64(), rateLimit.Flow.Inflow.Int64(), "inflow should be updated")
// }

// // TestOnRecvPacket_Denied tests the middleware's OnRecvPacket when the packet is denied
// func (s *KeeperTestSuite) TestOnRecvPacket_Denied() {
// 	path := ibctesting.NewPath(s.chainA, s.chainB) // Use ibctesting
// 	path.Setup()

// 	// Create rate limit with zero quota for recv
// 	rateLimitDenom := hashDenomTrace(fmt.Sprintf("%s/%s/%s", transferPort, path.EndpointB.ChannelID, uosmo))
// 	s.chainB.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainB.GetContext(), types.RateLimit{
// 		Path: &types.Path{Denom: rateLimitDenom, ChannelOrClientId: path.EndpointB.ChannelID},
// 		Quota: &types.Quota{MaxPercentRecv: sdkmath.ZeroInt(), DurationHours: 1}, // Zero quota
// 		Flow:  &types.Flow{Inflow: sdkmath.ZeroInt(), Outflow: sdkmath.ZeroInt(), ChannelValue: sdkmath.NewInt(1000)},
// 	})

// 	// Create packet data
// 	packetDataBz, err := json.Marshal(transfertypes.FungibleTokenPacketData{Denom: uosmo, Amount: "10"})
// 	s.Require().NoError(err)
// 	packet := channeltypes.Packet{
// 		Sequence:           1,
// 		SourcePort:         path.EndpointA.ChannelConfig.PortID,
// 		SourceChannel:      path.EndpointA.ChannelID,
// 		DestinationPort:    path.EndpointB.ChannelConfig.PortID,
// 		DestinationChannel: path.EndpointB.ChannelID,
// 		Data:               packetDataBz, // Correct variable
// 		TimeoutHeight:      clienttypes.ZeroHeight(),
// 		TimeoutTimestamp:   uint64(s.coordinator.CurrentTime.Add(time.Hour).UnixNano()),
// 	}

// 	// Relay the packet. This will call OnRecvPacket on chain B through the integrated middleware stack.
// 	err = path.RelayPacket(packet)
// 	s.Require().NoError(err, "relaying packet failed") // Check relay error first

// 	// Check acknowledgement on chain B
// 	ackBytes, found := s.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetPacketAcknowledgement(s.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
// 	s.Require().True(found, "acknowledgement not found")
// 	s.Require().NotNil(ackBytes, "ack bytes should not be nil")

// 	// Check acknowledgement is error
// 	var ack channeltypes.Acknowledgement
// 	// Use transfertypes codec to unmarshal
// 	err = transfertypes.ModuleCdc.UnmarshalJSON(ackBytes, &ack)
// 	s.Require().NoError(err, "unmarshalling acknowledgement failed")
// 	s.Require().False(ack.Success(), "ack should be error")

// 	// Check for the specific error within the acknowledgement data
// 	expectedAck := channeltypes.NewErrorAcknowledgement(types.ErrQuotaExceeded)
// 	s.Require().Equal(expectedAck.Acknowledgement(), ack.Acknowledgement(), "ack error message should match quota exceeded")

// 	// Check flow was NOT updated
// 	rateLimit, found := s.chainB.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainB.GetContext(), rateLimitDenom, path.EndpointB.ChannelID)
// 	s.Require().True(found)
// 	s.Require().True(rateLimit.Flow.Inflow.IsZero(), "inflow should NOT be updated")
// }

// // TestSendPacket_Allowed tests the middleware's SendPacket when the packet is allowed
// func (s *KeeperTestSuite) TestSendPacket_Allowed() {
// 	path := ibctesting.NewPath(s.chainA, s.chainB)
// 	path.Setup()

// 	// Create rate limit with sufficient quota
// 	rateLimitDenom := ustrd // Native denom
// 	s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{
// 		Path: &types.Path{Denom: rateLimitDenom, ChannelOrClientId: path.EndpointA.ChannelID},
// 		Quota: &types.Quota{MaxPercentSend: sdkmath.NewInt(100), DurationHours: 1}, // High quota
// 		Flow:  &types.Flow{Inflow: sdkmath.ZeroInt(), Outflow: sdkmath.ZeroInt(), ChannelValue: sdkmath.NewInt(1000)},
// 	})

// 	timeoutTimestamp := uint64(s.coordinator.CurrentTime.Add(time.Hour).UnixNano())

// 	// Create MsgTransfer
// 	transferMsg := transfertypes.NewMsgTransfer( // Use transfertypes
// 		path.EndpointA.ChannelConfig.PortID,
// 		path.EndpointA.ChannelID,
// 		sdk.NewCoin(ustrd, sdkmath.NewInt(10)),
// 		s.chainA.SenderAccount.GetAddress().String(),
// 		s.chainB.SenderAccount.GetAddress().String(),
// 		clienttypes.ZeroHeight(),
// 		timeoutTimestamp,
// 		"", // memo
// 	)

// 	// Send the message. This will invoke the transfer keeper, which calls SendPacket through the middleware stack.
// 	res, err := s.chainA.SendMsgs(transferMsg)
// 	s.Require().NoError(err)
// 	s.Require().NotNil(res)

// 	// Extract sequence from events
// 	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
// 	s.Require().NoError(err)
// 	seq := packet.Sequence
// 	s.Require().Equal(uint64(1), seq, "sequence should be 1")

// 	// Check flow was updated
// 	rateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), rateLimitDenom, path.EndpointA.ChannelID)
// 	s.Require().True(found)
// 	s.Require().Equal(sdkmath.NewInt(10).Int64(), rateLimit.Flow.Outflow.Int64(), "outflow should be updated")

// 	// Check pending packet was stored
// 	found = s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), path.EndpointA.ChannelID, seq)
// 	s.Require().True(found, "pending packet should be stored")
// }

// // TestSendPacket_Denied tests the middleware's SendPacket when the packet is denied
// func (s *KeeperTestSuite) TestSendPacket_Denied() {
// 	path := ibctesting.NewPath(s.chainA, s.chainB)
// 	path.Setup()

// 	// Create rate limit with zero quota
// 	rateLimitDenom := ustrd // Native denom
// 	s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{
// 		Path: &types.Path{Denom: rateLimitDenom, ChannelOrClientId: path.EndpointA.ChannelID},
// 		Quota: &types.Quota{MaxPercentSend: sdkmath.ZeroInt(), DurationHours: 1}, // Zero quota
// 		Flow:  &types.Flow{Inflow: sdkmath.ZeroInt(), Outflow: sdkmath.ZeroInt(), ChannelValue: sdkmath.NewInt(1000)},
// 	})

// 	timeoutTimestamp := uint64(s.coordinator.CurrentTime.Add(time.Hour).UnixNano())

// 	// Create MsgTransfer
// 	transferMsg := transfertypes.NewMsgTransfer(
// 		path.EndpointA.ChannelConfig.PortID,
// 		path.EndpointA.ChannelID,
// 		sdk.NewCoin(ustrd, sdkmath.NewInt(10)),
// 		s.chainA.SenderAccount.GetAddress().String(),
// 		s.chainB.SenderAccount.GetAddress().String(),
// 		clienttypes.ZeroHeight(),
// 		timeoutTimestamp,
// 		"", // memo
// 	)

// 	// Send the message. This will invoke the transfer keeper, which calls SendPacket through the middleware stack.
// 	_, err := s.chainA.SendMsgs(transferMsg)

// 	// Check error is quota exceeded
// 	s.Require().ErrorIs(err, types.ErrQuotaExceeded)

// 	// Check flow was NOT updated
// 	rateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), rateLimitDenom, path.EndpointA.ChannelID)
// 	s.Require().True(found)
// 	s.Require().True(rateLimit.Flow.Outflow.IsZero(), "outflow should NOT be updated")
// }
