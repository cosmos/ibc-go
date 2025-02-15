package types_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

var validPacketID = channeltypes.NewPacketID(ibctesting.MockFeePort, ibctesting.FirstChannelID, 1)

func TestKeyPayee(t *testing.T) {
	key := types.KeyPayee("relayer-address", ibctesting.FirstChannelID)
	require.Equal(t, string(key), fmt.Sprintf("%s/%s/%s", types.PayeeKeyPrefix, "relayer-address", ibctesting.FirstChannelID))
}

func TestParseKeyPayee(t *testing.T) {
	testCases := []struct {
		name   string
		key    string
		expErr error
	}{
		{
			"success",
			string(types.KeyPayee("relayer-address", ibctesting.FirstChannelID)),
			nil,
		},
		{
			"incorrect key - key split has incorrect length",
			"payeeAddress/relayer_address/transfer/channel-0",
			ibcerrors.ErrLogic,
		},
	}

	for _, tc := range testCases {
		tc := tc

		address, channelID, err := types.ParseKeyPayeeAddress(tc.key)

		if tc.expErr == nil {
			require.NoError(t, err)
			require.Equal(t, "relayer-address", address)
			require.Equal(t, ibctesting.FirstChannelID, channelID)
		} else {
			require.ErrorIs(t, err, tc.expErr)
		}
	}
}

func TestKeyCounterpartyPayee(t *testing.T) {
	var (
		relayerAddress = "relayer_address"
		channelID      = "channel-0"
	)

	key := types.KeyCounterpartyPayee(relayerAddress, channelID)
	require.Equal(t, string(key), fmt.Sprintf("%s/%s/%s", types.CounterpartyPayeeKeyPrefix, relayerAddress, channelID))
}

func TestKeyFeesInEscrow(t *testing.T) {
	key := types.KeyFeesInEscrow(validPacketID)
	require.Equal(t, string(key), fmt.Sprintf("%s/%s/%s/%d", types.FeesInEscrowPrefix, ibctesting.MockFeePort, ibctesting.FirstChannelID, 1))
}

func TestParseKeyFeeEnabled(t *testing.T) {
	testCases := []struct {
		name   string
		key    string
		expErr error
	}{
		{
			"success",
			string(types.KeyFeeEnabled(ibctesting.MockPort, ibctesting.FirstChannelID)),
			nil,
		},
		{
			"incorrect key - key split has incorrect length",
			string(types.KeyFeesInEscrow(validPacketID)),
			ibcerrors.ErrLogic,
		},
		{
			"incorrect key - key split has incorrect length",
			fmt.Sprintf("%s/%s/%s", "fee", ibctesting.MockPort, ibctesting.FirstChannelID),
			ibcerrors.ErrLogic,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			portID, channelID, err := types.ParseKeyFeeEnabled(tc.key)

			if tc.expErr == nil {
				require.NoError(t, err)
				require.Equal(t, ibctesting.MockPort, portID)
				require.Equal(t, ibctesting.FirstChannelID, channelID)
			} else {
				require.ErrorIs(t, err, tc.expErr)
				require.Empty(t, portID)
				require.Empty(t, channelID)
			}
		})
	}
}

func TestParseKeyFeesInEscrow(t *testing.T) {
	testCases := []struct {
		name   string
		key    string
		expErr error
	}{
		{
			"success",
			string(types.KeyFeesInEscrow(validPacketID)),
			nil,
		},
		{
			"incorrect key - key split has incorrect length",
			string(types.KeyFeeEnabled(validPacketID.PortId, validPacketID.ChannelId)),
			ibcerrors.ErrLogic,
		},
		{
			"incorrect key - sequence cannot be parsed",
			fmt.Sprintf("%s/%s", types.KeyFeesInEscrowChannelPrefix(validPacketID.PortId, validPacketID.ChannelId), "sequence"),
			errors.New("invalid syntax"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		packetID, err := types.ParseKeyFeesInEscrow(tc.key)

		if tc.expErr == nil {
			require.NoError(t, err)
			require.Equal(t, validPacketID, packetID)
		} else {
			ibctesting.RequireErrorIsOrContains(t, err, tc.expErr, err.Error())
		}
	}
}

func TestParseKeyForwardRelayerAddress(t *testing.T) {
	testCases := []struct {
		name   string
		key    string
		expErr error
	}{
		{
			"success",
			string(types.KeyRelayerAddressForAsyncAck(validPacketID)),
			nil,
		},
		{
			"incorrect key - key split has incorrect length",
			"forwardRelayer/transfer/channel-0",
			ibcerrors.ErrLogic,
		},
		{
			"incorrect key - sequence is not correct",
			"forwardRelayer/transfer/channel-0/sequence",
			errors.New("invalid syntax"),
		},
	}

	for _, tc := range testCases {
		tc := tc

		packetID, err := types.ParseKeyRelayerAddressForAsyncAck(tc.key)

		if tc.expErr == nil {
			require.NoError(t, err)
			require.Equal(t, validPacketID, packetID)
		} else {
			ibctesting.RequireErrorIsOrContains(t, err, tc.expErr, err.Error())
		}
	}
}

func TestParseKeyCounterpartyPayee(t *testing.T) {
	relayerAddress := "relayer_address"

	testCases := []struct {
		name   string
		key    string
		expErr error
	}{
		{
			"success",
			string(types.KeyCounterpartyPayee(relayerAddress, ibctesting.FirstChannelID)),
			nil,
		},
		{
			"incorrect key - key split has incorrect length",
			"relayerAddress/relayer_address/transfer/channel-0",
			ibcerrors.ErrLogic,
		},
	}

	for _, tc := range testCases {
		tc := tc

		address, channelID, err := types.ParseKeyCounterpartyPayee(tc.key)

		if tc.expErr == nil {
			require.NoError(t, err)
			require.Equal(t, relayerAddress, address)
			require.Equal(t, ibctesting.FirstChannelID, channelID)
		} else {
			require.ErrorIs(t, err, tc.expErr)
		}
	}
}
