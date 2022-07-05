package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

var (
	addr1       = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	addr2       = sdk.AccAddress("testaddr2").String()
	validCoins  = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	validCoins2 = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(200)}}
	validCoins3 = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(300)}}
)

func TestValidateGenesis(t *testing.T) {
	var (
		packetID        channeltypes.PacketId
		fee             types.Fee
		refundAcc       string
		sender          string
		forwardAddr     string
		counterparty    string
		portID          string
		channelID       string
		packetChannelID string
		seq             uint64
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"valid genesis",
			func() {},
			true,
		},
		{
			"invalid packetID: invalid channel",
			func() {
				packetID = channeltypes.NewPacketId(
					portID,
					"",
					seq,
				)
			},
			false,
		},
		{
			"invalid packetID: invalid port",
			func() {
				packetID = channeltypes.NewPacketId(
					"",
					channelID,
					seq,
				)
			},
			false,
		},
		{
			"invalid packetID: invalid sequence",
			func() {
				packetID = channeltypes.NewPacketId(
					portID,
					channelID,
					0,
				)
			},
			false,
		},
		{
			"invalid packetID: invalid fee",
			func() {
				fee = types.Fee{
					sdk.Coins{},
					sdk.Coins{},
					sdk.Coins{},
				}
			},
			false,
		},
		{
			"invalid packetID: invalid refundAcc",
			func() {
				refundAcc = ""
			},
			false,
		},
		{
			"invalid FeeEnabledChannel: invalid ChannelID",
			func() {
				channelID = ""
			},
			false,
		},
		{
			"invalid FeeEnabledChannel: invalid PortID",
			func() {
				portID = ""
			},
			false,
		},
		{
			"invalid RegisteredRelayers: invalid sender",
			func() {
				sender = ""
			},
			false,
		},
		{
			"invalid RegisteredRelayers: invalid counterparty",
			func() {
				counterparty = " "
			},
			false,
		},
		{
			"invalid ForwardRelayerAddress: invalid forwardAddr",
			func() {
				forwardAddr = ""
			},
			false,
		},
		{
			"invalid ForwardRelayerAddress: invalid packet",
			func() {
				packetChannelID = "1"
			},
			false,
		},
	}

	for _, tc := range testCases {
		portID = transfertypes.PortID
		channelID = ibctesting.FirstChannelID
		packetChannelID = ibctesting.FirstChannelID
		seq = uint64(1)

		// build PacketId & Fee
		packetID = channeltypes.NewPacketId(
			portID,
			channelID,
			seq,
		)
		fee = types.Fee{
			validCoins,
			validCoins2,
			validCoins3,
		}

		refundAcc = addr1

		// relayer addresses
		sender = addr1
		counterparty = addr2
		forwardAddr = addr2

		tc.malleate()

		genState := types.GenesisState{
			IdentifiedFees: []types.IdentifiedPacketFees{
				{
					PacketId: packetID,
					PacketFees: []types.PacketFee{
						{
							Fee:           fee,
							RefundAddress: refundAcc,
							Relayers:      nil,
						},
					},
				},
			},
			FeeEnabledChannels: []types.FeeEnabledChannel{
				{
					PortId:    portID,
					ChannelId: channelID,
				},
			},
			RegisteredRelayers: []types.RegisteredRelayerAddress{
				{
					Address:             sender,
					CounterpartyAddress: counterparty,
				},
			},
			ForwardRelayers: []types.ForwardRelayerAddress{
				{
					Address:  forwardAddr,
					PacketId: channeltypes.NewPacketId(portID, packetChannelID, 1),
				},
			},
		}

		err := genState.Validate()
		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}

func TestValidateDefaultGenesis(t *testing.T) {
	err := types.DefaultGenesisState().Validate()
	require.NoError(t, err)
}
