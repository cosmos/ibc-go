package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

var (
	validChannelID = "channel-1"
	validPortID    = "validPortId"
	invalidID      = "this identifier is too long to be used as a valid identifier"
	validCoins     = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	validAddr      = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	invalidAddr    = "invalid_address"
)

// TestMsgTransferValidation tests ValidateBasic for MsgTransfer
func TestMsgRegisterCountepartyAddressValidation(t *testing.T) {
	testCases := []struct {
		name    string
		msg     *MsgRegisterCounterpartyAddress
		expPass bool
	}{
		{"validate with correct sdk.AccAddress", NewMsgRegisterCounterpartyAddress(validAddr, validAddr), true},
		{"validate with incorrect destination relayer address", NewMsgRegisterCounterpartyAddress(invalidAddr, validAddr), false},
		{"validate with incorrect counterparty relayer address", NewMsgRegisterCounterpartyAddress(validAddr, invalidAddr), false},
	}

	for i, tc := range testCases {
		err := tc.msg.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

// TestRegisterCounterpartyAddressGetSigners tests GetSigners
func TestRegisterCountepartyAddressGetSigners(t *testing.T) {
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	// build message
	msg := NewMsgRegisterCounterpartyAddress(addr.String(), addr.String())

	// GetSigners
	res := msg.GetSigners()

	require.Equal(t, []sdk.AccAddress{addr}, res)
}

// TestMsgPayPacketFeeValidation tests ValidateBasic
func TestMsgPayPacketFeeValidation(t *testing.T) {
	var (
		signer    string
		channelID string
		portID    string
		fee       Fee
		relayers  []string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"invalid channelID",
			func() {
				channelID = invalidID
			},
			false,
		},
		{
			"invalid portID",
			func() {
				portID = invalidID
			},
			false,
		},
		{
			"relayers is not nil",
			func() {
				relayers = []string{validAddr}
			},
			false,
		},
		{
			"invalid signer address",
			func() {
				signer = "invalid-addr"
			},
			false,
		},
	}

	for _, tc := range testCases {
		// build message
		signer = validAddr
		channelID = validChannelID
		portID = validPortID
		fee = Fee{validCoins, validCoins, validCoins}
		relayers = nil

		// malleate
		tc.malleate()
		msg := NewMsgPayPacketFee(fee, portID, channelID, signer, relayers)

		err := msg.ValidateBasic()

		if tc.expPass {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

// TestPayPacketFeeGetSigners tests GetSigners
func TestPayPacketFeeGetSigners(t *testing.T) {
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	// build message
	signer := addr.String()
	channelID := validChannelID
	portID := validPortID
	fee := Fee{validCoins, validCoins, validCoins}
	msg := NewMsgPayPacketFee(fee, portID, channelID, signer, nil)

	// GetSigners
	res := msg.GetSigners()

	require.Equal(t, []sdk.AccAddress{addr}, res)
}

// TestMsgPayPacketFeeAsyncValidation tests ValidateBasic
func TestMsgPayPacketFeeAsyncValidation(t *testing.T) {
	var (
		signer    string
		channelID string
		portID    string
		fee       Fee
		relayers  []string
		seq       uint64
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"invalid channelID",
			func() {
				channelID = invalidID
			},
			false,
		},
		{
			"invalid portID",
			func() {
				portID = invalidID
			},
			false,
		},
		{
			"relayers is not nil",
			func() {
				relayers = []string{validAddr}
			},
			false,
		},
		{
			"invalid signer address",
			func() {
				signer = "invalid-addr"
			},
			false,
		},
		{
			"invalid sequence",
			func() {
				seq = 0
			},
			false,
		},
	}

	for _, tc := range testCases {
		// build message
		signer = validAddr
		channelID = validChannelID
		portID = validPortID
		fee = Fee{validCoins, validCoins, validCoins}
		relayers = nil
		seq = 1

		// malleate
		tc.malleate()

		packetId := &channeltypes.PacketId{ChannelId: channelID, PortId: portID, Sequence: seq}
		identifiedPacketFee := IdentifiedPacketFee{PacketId: packetId, Fee: fee, Relayers: relayers}
		msg := NewMsgPayPacketFeeAsync(identifiedPacketFee, signer)

		err := msg.ValidateBasic()

		if tc.expPass {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}

// TestRegisterCounterpartyAddressGetSigners tests GetSigners
func TestPayPacketFeeAsyncGetSigners(t *testing.T) {
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	// build message
	channelID := validChannelID
	portID := validPortID
	fee := Fee{validCoins, validCoins, validCoins}
	seq := uint64(1)
	packetId := &channeltypes.PacketId{ChannelId: channelID, PortId: portID, Sequence: seq}
	identifiedPacketFee := IdentifiedPacketFee{PacketId: packetId, Fee: fee, Relayers: nil}
	msg := NewMsgPayPacketFeeAsync(identifiedPacketFee, addr.String())

	// GetSigners
	res := msg.GetSigners()

	require.Equal(t, []sdk.AccAddress{addr}, res)
}
