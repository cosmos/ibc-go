package keeper_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"

	ratelimiting "github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
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

func (s *KeeperTestSuite) TestCheckAcknowledgementSucceeded() {
	testCases := []struct {
		name        string
		ack         []byte
		wantSuccess bool
		wantErr     error
	}{
		{
			name: "success legacy format",
			ack: func() []byte {
				return transfertypes.ModuleCdc.MustMarshalJSON(&channeltypes.Acknowledgement{
					Response: &channeltypes.Acknowledgement_Result{Result: []byte{1}},
				})
			}(),
			wantSuccess: true,
			wantErr:     nil,
		},
		{
			name: "failed legacy format - empty result",
			ack: func() []byte {
				return transfertypes.ModuleCdc.MustMarshalJSON(&channeltypes.Acknowledgement{
					Response: &channeltypes.Acknowledgement_Result{},
				})
			}(),
			wantSuccess: false,
			wantErr:     channeltypes.ErrInvalidAcknowledgement,
		},
		{
			name: "failed legacy format",
			ack: func() []byte {
				return transfertypes.ModuleCdc.MustMarshalJSON(&channeltypes.Acknowledgement{
					Response: &channeltypes.Acknowledgement_Error{Error: "some error"},
				})
			}(),
			wantSuccess: false,
			wantErr:     nil,
		},
		{
			name:        "failed v2 format",
			ack:         channeltypesv2.ErrorAcknowledgement[:],
			wantSuccess: false,
			wantErr:     nil,
		},
		{
			name:        "invalid format",
			ack:         []byte("invalid ack"),
			wantSuccess: false,
			wantErr:     sdkerrors.ErrUnknownRequest,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			success, err := s.chainA.GetSimApp().RateLimitKeeper.CheckAcknowledementSucceeded(s.chainA.GetContext(), tc.ack)

			if tc.wantErr != nil {
				s.Require().ErrorIs(err, tc.wantErr, tc.name)
			} else {
				s.Require().NoError(err, "unexpected error for %s", tc.name)
			}

			s.Require().Equal(tc.wantSuccess, success,
				"expected success=%v for %s", tc.wantSuccess, tc.name)
		})
	}
}

func (s *KeeperTestSuite) createRateLimitCloseToQuota(denom string, channelID string, direction types.PacketDirection) {
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
			ChannelOrClientId: channelID,
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
	amountToExceed := "5"
	sequence := uint64(10)

	// Create rate limit (for SEND, use SOURCE channel)
	s.createRateLimitCloseToQuota(denom, sourceChannel, types.PACKET_SEND)

	// This packet should cause an Outflow quota exceed error
	packetData, err := json.Marshal(transfertypes.FungibleTokenPacketData{Denom: denom, Amount: amountToExceed})
	s.Require().NoError(err)

	s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetNextSequenceSend(s.chainA.GetContext(), transferPort, sourceChannel, sequence)
	// We check for a quota error because it doesn't appear until the end of the function
	// We're avoiding checking for a success here because we can get a false positive if the rate limit doesn't exist
	err = s.chainA.GetSimApp().RateLimitKeeper.SendRateLimitedPacket(s.chainA.GetContext(), transferPort, sourceChannel, clienttypes.Height{}, 0, packetData)
	s.Require().ErrorIs(err, types.ErrQuotaExceeded, "error type")
	s.Require().ErrorContains(err, "Outflow exceeds quota", "error text")

	// Reset the rate limit and try again
	err = s.chainA.GetSimApp().RateLimitKeeper.ResetRateLimit(s.chainA.GetContext(), denom, channelID)
	s.Require().NoError(err, "no error expected when resetting rate limit")

	err = s.chainA.GetSimApp().RateLimitKeeper.SendRateLimitedPacket(s.chainA.GetContext(), transferPort, sourceChannel, clienttypes.Height{}, 0, packetData)
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
		Path: &types.Path{Denom: denom, ChannelOrClientId: channelID},
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
		Path: &types.Path{Denom: denom, ChannelOrClientId: channelID},
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
		Path: &types.Path{Denom: denom, ChannelOrClientId: channelID},
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
	rateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denom, channelID)
	s.Require().True(found)
	s.Require().Equal(expectedOutflow.Int64(), rateLimit.Flow.Outflow.Int64(), "outflow decremented")

	// Check that the pending packet has been removed
	found = s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(s.chainA.GetContext(), channelID, sequence)
	s.Require().False(found, "pending packet should have been removed")

	// Call OnTimeoutPacket again with a different sequence number
	// (to simulate a timeout that arrived in a different quota window from where the send occurred)
	// The outflow should not change
	packet.Sequence--
	err = s.chainA.GetSimApp().RateLimitKeeper.TimeoutRateLimitedPacket(s.chainA.GetContext(), packet)
	s.Require().NoError(err, "no error expected when calling timeout packet again")

	rateLimit, found = s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainA.GetContext(), denom, channelID)
	s.Require().True(found)
	s.Require().Equal(expectedOutflow.Int64(), rateLimit.Flow.Outflow.Int64(), "outflow should not have changed")
}

// --- Middleware Tests ---

// TestOnRecvPacket_Allowed tests the middleware's OnRecvPacket when the packet is allowed
func (s *KeeperTestSuite) TestOnRecvPacket_Allowed() {
	path := ibctesting.NewTransferPath(s.chainA, s.chainB)
	path.Setup()

	// Define recipient and calculate expected voucher denom on chain B
	recipientAddr := s.chainB.SenderAccount.GetAddress()
	voucherDenomStr := hashDenomTrace(fmt.Sprintf("%s/%s/%s", transferPort, path.EndpointB.ChannelID, uosmo))

	// Fund recipient account with native denom
	fundAmount := sdkmath.NewInt(1000000)
	bondDenom, err := s.chainB.GetSimApp().StakingKeeper.BondDenom(s.chainB.GetContext())
	s.Require().NoError(err, "getting bond denom failed")
	fundCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, fundAmount))
	// Mint native denom to transfer module
	err = s.chainB.GetSimApp().BankKeeper.MintCoins(s.chainB.GetContext(), transfertypes.ModuleName, fundCoins)
	s.Require().NoError(err, "minting native denom coins to transfer module failed")
	// Send native denom from transfer module to recipient
	err = s.chainB.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(s.chainB.GetContext(), transfertypes.ModuleName, recipientAddr, fundCoins)
	s.Require().NoError(err, "funding recipient account with native denom failed")

	// Create the test packet data
	testAmountStr := "10"
	testAmountInt, _ := sdkmath.NewIntFromString(testAmountStr)
	packetDataBz, err := json.Marshal(transfertypes.FungibleTokenPacketData{
		Denom:    uosmo,
		Amount:   testAmountStr,
		Sender:   s.chainA.SenderAccount.GetAddress().String(),
		Receiver: recipientAddr.String(),
	})
	s.Require().NoError(err)

	// Set the rate limit using the voucher denom string
	simulatedSupply := sdkmath.NewInt(1000) // Keep simulated supply for rate limit calculation
	s.chainB.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainB.GetContext(), types.RateLimit{
		Path:  &types.Path{Denom: voucherDenomStr, ChannelOrClientId: path.EndpointB.ChannelID},
		Quota: &types.Quota{MaxPercentRecv: sdkmath.NewInt(100), DurationHours: 1}, // High quota
		Flow:  &types.Flow{Inflow: sdkmath.ZeroInt(), Outflow: sdkmath.ZeroInt(), ChannelValue: simulatedSupply},
	})

	timeoutTS := uint64(s.coordinator.CurrentTime.Add(time.Hour).UnixNano())
	// Commit the packet on chain A so that RelayPacket can find the commitment
	seq, err := path.EndpointA.SendPacket(clienttypes.ZeroHeight(), timeoutTS, packetDataBz)
	s.Require().NoError(err, "sending packet on chain A failed")

	packet := channeltypes.Packet{
		Sequence:           seq,
		SourcePort:         path.EndpointA.ChannelConfig.PortID,
		SourceChannel:      path.EndpointA.ChannelID,
		DestinationPort:    path.EndpointB.ChannelConfig.PortID,
		DestinationChannel: path.EndpointB.ChannelID,
		Data:               packetDataBz,
		TimeoutHeight:      clienttypes.ZeroHeight(),
		TimeoutTimestamp:   timeoutTS,
	}

	// Relay the packet. This will call OnRecvPacket on chain B through the integrated middleware stack.
	err = path.RelayPacket(packet)
	s.Require().NoError(err, "relaying packet failed")

	// Check acknowledgement on chain B
	ackBz, found := s.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetPacketAcknowledgement(s.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	s.Require().True(found, "acknowledgement not found")
	s.Require().NotNil(ackBz, "ack should not be nil")

	expectedAck := channeltypes.NewResultAcknowledgement([]byte{1})
	expBz := channeltypes.CommitAcknowledgement(expectedAck.Acknowledgement())
	s.Require().Equal(expBz, ackBz)

	// Check flow was updated
	rateLimit, found := s.chainB.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainB.GetContext(), voucherDenomStr, path.EndpointB.ChannelID)
	s.Require().True(found)
	s.Require().Equal(testAmountInt.Int64(), rateLimit.Flow.Inflow.Int64(), "inflow should be updated")
}

// TestOnRecvPacket_Denied tests the middleware's OnRecvPacket when the packet is denied
func (s *KeeperTestSuite) TestOnRecvPacket_Denied() {
	path := ibctesting.NewTransferPath(s.chainA, s.chainB)
	path.Setup()

	// Create rate limit with zero quota for recv
	rateLimitDenom := hashDenomTrace(fmt.Sprintf("%s/%s/%s", transferPort, path.EndpointB.ChannelID, sdk.DefaultBondDenom))
	s.chainB.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainB.GetContext(), types.RateLimit{
		Path:  &types.Path{Denom: rateLimitDenom, ChannelOrClientId: path.EndpointB.ChannelID},
		Quota: &types.Quota{MaxPercentRecv: sdkmath.ZeroInt(), DurationHours: 1}, // Zero quota
		Flow:  &types.Flow{Inflow: sdkmath.ZeroInt(), Outflow: sdkmath.ZeroInt(), ChannelValue: sdkmath.NewInt(1000)},
	})

	sender := s.chainA.SenderAccount.GetAddress()
	receiver := s.chainB.SenderAccount.GetAddress()
	sendCoin := ibctesting.TestCoin

	// Create packet data
	packetDataBz, err := json.Marshal(transfertypes.FungibleTokenPacketData{
		Denom:    sendCoin.Denom,
		Amount:   sendCoin.Amount.String(),
		Sender:   sender.String(),
		Receiver: receiver.String(),
	})
	s.Require().NoError(err)

	timeoutTS := uint64(s.coordinator.CurrentTime.Add(time.Hour).UnixNano())
	timeoutHeight := clienttypes.ZeroHeight()
	sourcePort := path.EndpointA.ChannelConfig.PortID
	sourceChannel := path.EndpointA.ChannelID
	senderInitialBal := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)

	// Commit the packet on chain A so that RelayPacket can find the commitment
	transferMsg := transfertypes.NewMsgTransfer(sourcePort, sourceChannel, sendCoin, sender.String(), receiver.String(), timeoutHeight, timeoutTS, "")
	resp, err := s.chainA.GetSimApp().TransferKeeper.Transfer(s.chainA.GetContext(), transferMsg)
	s.Require().NoError(err)

	// After sending the transfer, "sendCoin" should be taken from the sender to escrow.
	senderIntermedBal := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)
	s.Require().Equal(senderInitialBal.Sub(sendCoin), senderIntermedBal)

	// Manully commit block on Chain A
	s.coordinator.CommitBlock(s.chainA)

	packet := channeltypes.Packet{
		Sequence:           resp.Sequence,
		SourcePort:         sourcePort,
		SourceChannel:      sourceChannel,
		DestinationPort:    path.EndpointB.ChannelConfig.PortID,
		DestinationChannel: path.EndpointB.ChannelID,
		Data:               packetDataBz,
		TimeoutHeight:      timeoutHeight,
		TimeoutTimestamp:   timeoutTS,
	}

	// Relay the packet. This will call OnRecvPacket on chain B through the integrated middleware stack.
	err = path.RelayPacket(packet)
	s.Require().NoError(err, "relaying packet failed")

	// Check acknowledgement on chain B
	ackBytes, found := s.chainB.GetSimApp().IBCKeeper.ChannelKeeper.GetPacketAcknowledgement(s.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	s.Require().True(found, "acknowledgement not found")
	s.Require().NotNil(ackBytes, "ack bytes should not be nil")

	expectedAck := channeltypes.NewErrorAcknowledgement(types.ErrQuotaExceeded)
	expBz := channeltypes.CommitAcknowledgement(expectedAck.Acknowledgement())
	s.Require().Equal(expBz, ackBytes)

	// Check flow was NOT updated
	rateLimit, found := s.chainB.GetSimApp().RateLimitKeeper.GetRateLimit(s.chainB.GetContext(), rateLimitDenom, path.EndpointB.ChannelID)
	s.Require().True(found)
	s.Require().True(rateLimit.Flow.Inflow.IsZero(), "inflow should NOT be updated")

	// Sender should be refunded
	senderEndBal := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), sender, sdk.DefaultBondDenom)
	s.Require().Equal(senderInitialBal, senderEndBal)
}

// TestSendPacket_Allowed tests the middleware's SendPacket when the packet is allowed by directly calling the middleware
func (s *KeeperTestSuite) TestSendPacket_Allowed() {
	path := ibctesting.NewTransferPath(s.chainA, s.chainB)
	path.Setup()

	// Create rate limit with sufficient quota
	rateLimitDenom := ustrd // Native denom
	s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{
		Path:  &types.Path{Denom: rateLimitDenom, ChannelOrClientId: path.EndpointA.ChannelID},
		Quota: &types.Quota{MaxPercentSend: sdkmath.NewInt(100), DurationHours: 1}, // High quota
		Flow:  &types.Flow{Inflow: sdkmath.ZeroInt(), Outflow: sdkmath.ZeroInt(), ChannelValue: sdkmath.NewInt(1000)},
	})

	timeoutTimestamp := uint64(s.coordinator.CurrentTime.Add(time.Hour).UnixNano())
	amount := sdkmath.NewInt(10)

	// Create packet data
	packetData := transfertypes.FungibleTokenPacketData{
		Denom:    ustrd,
		Amount:   amount.String(),
		Sender:   s.chainA.SenderAccount.GetAddress().String(),
		Receiver: s.chainB.SenderAccount.GetAddress().String(),
		Memo:     "",
	}
	packetDataBz, err := json.Marshal(packetData)
	s.Require().NoError(err)

	// We need the transfer keeper's ICS4Wrapper which *is* the ratelimiting middleware
	middleware, ok := s.chainA.GetSimApp().PFMKeeper.ICS4Wrapper().(*ratelimiting.IBCMiddleware)
	s.Require().Truef(ok, "PFM's ICS4Wrapper should be the Rate Limit Middleware. Found %T", s.chainA.GetSimApp().TransferKeeper.GetICS4Wrapper())

	// Directly call the middleware's SendPacket
	seq, err := middleware.SendPacket(
		s.chainA.GetContext(),
		path.EndpointA.ChannelConfig.PortID,
		path.EndpointA.ChannelID,
		clienttypes.ZeroHeight(), // timeout height
		timeoutTimestamp,
		packetDataBz,
	)

	// Assert SendPacket succeeded
	s.Require().NoError(err, "middleware.SendPacket should succeed")
	s.Require().Equal(uint64(1), seq, "sequence should be 1")

	// Commit block and update context to ensure state updates are visible
	s.coordinator.CommitBlock(s.chainA)
	ctx := s.chainA.GetContext() // Get the latest context after commit

	// Check flow was updated using the latest context
	rateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(ctx, rateLimitDenom, path.EndpointA.ChannelID)
	s.Require().True(found)
	s.Require().Equal(amount.Int64(), rateLimit.Flow.Outflow.Int64(), "outflow should be updated")

	// Check pending packet was stored using the latest context
	found = s.chainA.GetSimApp().RateLimitKeeper.CheckPacketSentDuringCurrentQuota(ctx, path.EndpointA.ChannelID, seq)
	s.Require().True(found, "pending packet should be stored")
}

// TestSendPacket_Denied tests the middleware's SendPacket when the packet is denied by directly calling the middleware
func (s *KeeperTestSuite) TestSendPacket_Denied() {
	path := ibctesting.NewTransferPath(s.chainA, s.chainB)
	path.Setup()

	// Create rate limit with a tiny quota that will be exceeded
	rateLimitDenom := ustrd // Native denom
	s.chainA.GetSimApp().RateLimitKeeper.SetRateLimit(s.chainA.GetContext(), types.RateLimit{
		Path:  &types.Path{Denom: rateLimitDenom, ChannelOrClientId: path.EndpointA.ChannelID},
		Quota: &types.Quota{MaxPercentSend: sdkmath.NewInt(1), DurationHours: 1}, // Set quota to 1% (will allow < 10 with ChannelValue 1000)
		Flow:  &types.Flow{Inflow: sdkmath.ZeroInt(), Outflow: sdkmath.ZeroInt(), ChannelValue: sdkmath.NewInt(1000)},
	})

	timeoutTimestamp := uint64(s.coordinator.CurrentTime.Add(time.Hour).UnixNano())
	amount := sdkmath.NewInt(11) // amount 11 will exceed 1% of 1000 (threshold is 10, check is GT)

	// Create packet data
	packetData := transfertypes.FungibleTokenPacketData{
		Denom:    ustrd,
		Amount:   amount.String(),
		Sender:   s.chainA.SenderAccount.GetAddress().String(),
		Receiver: s.chainB.SenderAccount.GetAddress().String(),
		Memo:     "",
	}
	packetDataBz, err := json.Marshal(packetData)
	s.Require().NoError(err)

	// Get the middleware instance
	middleware, ok := s.chainA.GetSimApp().PFMKeeper.ICS4Wrapper().(*ratelimiting.IBCMiddleware)
	s.Require().Truef(ok, "Packet forward middleware keeper's ICS4Wrapper should be the RateLimit middleware. Found: %T", middleware)

	// Directly call the middleware's SendPacket
	_, err = middleware.SendPacket(
		s.chainA.GetContext(),
		path.EndpointA.ChannelConfig.PortID,
		path.EndpointA.ChannelID,
		clienttypes.ZeroHeight(), // timeout height
		timeoutTimestamp,
		packetDataBz,
	)

	// Check error is quota exceeded
	s.Require().Error(err, "middleware.SendPacket should fail")
	s.Require().ErrorIs(err, types.ErrQuotaExceeded, "error should be quota exceeded")

	// Commit block and update context
	s.coordinator.CommitBlock(s.chainA)
	ctx := s.chainA.GetContext() // Get latest context

	// Check flow was NOT updated
	rateLimit, found := s.chainA.GetSimApp().RateLimitKeeper.GetRateLimit(ctx, rateLimitDenom, path.EndpointA.ChannelID)
	s.Require().True(found)
	s.Require().True(rateLimit.Flow.Outflow.IsZero(), "outflow should NOT be updated")
}
