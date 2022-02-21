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
		packetId     channeltypes.PacketId
		fee          types.Fee
		refundAcc    string
		sender       string
		counterparty string
		portID       string
		channelID    string
		seq          uint64
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
			"invalid packetId: invalid channel",
			func() {
				packetId = channeltypes.NewPacketId(
					"",
					portID,
					seq,
				)
			},
			false,
		},
		{
			"invalid packetId: invalid port",
			func() {
				packetId = channeltypes.NewPacketId(
					channelID,
					"",
					seq,
				)
			},
			false,
		},
		{
			"invalid packetId: invalid sequence",
			func() {
				packetId = channeltypes.NewPacketId(
					channelID,
					portID,
					0,
				)
			},
			false,
		},
		{
			"invalid packetId: invalid fee",
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
			"invalid packetId: invalid refundAcc",
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
				counterparty = ""
			},
			false,
		},
	}

	for _, tc := range testCases {
		portID = transfertypes.PortID
		channelID = ibctesting.FirstChannelID
		seq = uint64(1)

		// build PacketId & Fee
		packetId = channeltypes.NewPacketId(
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

		tc.malleate()

		genState := types.GenesisState{
			IdentifiedFees: []types.IdentifiedPacketFees{
				{
					PacketId: packetId,
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
