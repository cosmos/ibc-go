package types_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func TestValidateDefaultGenesis(t *testing.T) {
	err := types.DefaultGenesisState().Validate()
	require.NoError(t, err)
}

func TestValidateGenesis(t *testing.T) {
	var genState *types.GenesisState

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success - valid genesis",
			func() {},
			nil,
		},
		{
			"invalid packetID: invalid port ID",
			func() {
				genState.IdentifiedFees[0].PacketId = channeltypes.NewPacketID("", ibctesting.FirstChannelID, 1)
			},
			host.ErrInvalidID,
		},
		{
			"invalid packetID: invalid channel ID",
			func() {
				genState.IdentifiedFees[0].PacketId = channeltypes.NewPacketID(ibctesting.MockFeePort, "", 1)
			},
			host.ErrInvalidID,
		},
		{
			"invalid packetID: invalid sequence",
			func() {
				genState.IdentifiedFees[0].PacketId = channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 0)
			},
			host.ErrInvalidPacket,
		},
		{
			"invalid packet fee: invalid fee",
			func() {
				genState.IdentifiedFees[0].PacketFees[0].Fee = types.NewFee(sdk.Coins{}, sdk.Coins{}, sdk.Coins{})
			},
			ibcerrors.ErrInvalidCoins,
		},
		{
			"invalid packet fee: invalid refund address",
			func() {
				genState.IdentifiedFees[0].PacketFees[0].RefundAddress = ""
			},
			errors.New("failed to convert RefundAddress into sdk.AccAddress"),
		},
		{
			"invalid fee enabled channel: invalid port ID",
			func() {
				genState.FeeEnabledChannels[0].PortId = ""
			},
			host.ErrInvalidID,
		},
		{
			"invalid fee enabled channel: invalid channel ID",
			func() {
				genState.FeeEnabledChannels[0].ChannelId = ""
			},
			host.ErrInvalidID,
		},
		{
			"invalid registered payee: invalid relayer address",
			func() {
				genState.RegisteredPayees[0].Relayer = ""
			},
			errors.New("failed to convert relayer address into sdk.AccAddress"),
		},
		{
			"invalid registered payee: invalid payee address",
			func() {
				genState.RegisteredPayees[0].Payee = ""
			},
			errors.New("failed to convert payee address into sdk.AccAddress"),
		},
		{
			"invalid registered payee: invalid channel ID",
			func() {
				genState.RegisteredPayees[0].ChannelId = ""
			},
			host.ErrInvalidID,
		},
		{
			"invalid registered counterparty payees: invalid relayer address",
			func() {
				genState.RegisteredCounterpartyPayees[0].Relayer = ""
			},
			errors.New("failed to convert relayer address into sdk.AccAddress"),
		},
		{
			"invalid registered counterparty payees: invalid counterparty payee",
			func() {
				genState.RegisteredCounterpartyPayees[0].CounterpartyPayee = ""
			},
			types.ErrCounterpartyPayeeEmpty,
		},
		{
			"invalid forward relayer address: invalid forward address",
			func() {
				genState.ForwardRelayers[0].Address = ""
			},
			errors.New("failed to convert forward relayer address into sdk.AccAddress"),
		},
		{
			"invalid forward relayer address: invalid packet",
			func() {
				genState.ForwardRelayers[0].PacketId = channeltypes.PacketId{}
			},
			host.ErrInvalidID,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			genState = &types.GenesisState{
				IdentifiedFees: []types.IdentifiedPacketFees{
					{
						PacketId:   channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1),
						PacketFees: []types.PacketFee{types.NewPacketFee(types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee), defaultAccAddress, nil)},
					},
				},
				FeeEnabledChannels: []types.FeeEnabledChannel{
					{
						PortId:    ibctesting.MockFeePort,
						ChannelId: ibctesting.FirstChannelID,
					},
				},
				RegisteredCounterpartyPayees: []types.RegisteredCounterpartyPayee{
					{
						Relayer:           defaultAccAddress,
						CounterpartyPayee: defaultAccAddress,
						ChannelId:         ibctesting.FirstChannelID,
					},
				},
				ForwardRelayers: []types.ForwardRelayerAddress{
					{
						Address:  defaultAccAddress,
						PacketId: channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1),
					},
				},
				RegisteredPayees: []types.RegisteredPayee{
					{
						Relayer:   sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String(),
						Payee:     sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String(),
						ChannelId: ibctesting.FirstChannelID,
					},
				},
			}

			tc.malleate()

			err := genState.Validate()

			if tc.expErr == nil {
				require.NoError(t, err, tc.name)
			} else {
				ibctesting.RequireErrorIsOrContains(t, err, tc.expErr, err.Error())
			}
		})
	}
}
