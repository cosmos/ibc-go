package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

func TestChannelValidateBasic(t *testing.T) {
	counterparty := types.Counterparty{"portidone", "channelidone"}
	testCases := []struct {
		name    string
		channel types.Channel
		expErr  error
	}{
		{"valid channel", types.NewChannel(types.TRYOPEN, types.ORDERED, counterparty, connHops, version), nil},
		{"invalid state", types.NewChannel(types.UNINITIALIZED, types.ORDERED, counterparty, connHops, version), types.ErrInvalidChannelState},
		{"invalid order", types.NewChannel(types.TRYOPEN, types.NONE, counterparty, connHops, version), types.ErrInvalidChannelOrdering},
		{"more than 1 connection hop", types.NewChannel(types.TRYOPEN, types.ORDERED, counterparty, []string{"connection1", "connection2"}, version), types.ErrTooManyConnectionHops},
		{"invalid connection hop identifier", types.NewChannel(types.TRYOPEN, types.ORDERED, counterparty, []string{"(invalid)"}, version), host.ErrInvalidID},
		{"invalid counterparty", types.NewChannel(types.TRYOPEN, types.ORDERED, types.NewCounterparty("(invalidport)", "channelidone"), connHops, version), host.ErrInvalidID},
	}

	for i, tc := range testCases {
		err := tc.channel.ValidateBasic()
		if tc.expErr == nil {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
			require.ErrorIs(t, err, tc.expErr)
		}
	}
}

func TestCounterpartyValidateBasic(t *testing.T) {
	testCases := []struct {
		name         string
		counterparty types.Counterparty
		expErr       error
	}{
		{"valid counterparty", types.Counterparty{"portidone", "channelidone"}, nil},
		{"invalid port id", types.Counterparty{"(InvalidPort)", "channelidone"}, host.ErrInvalidID},
		{"invalid channel id", types.Counterparty{"portidone", "(InvalidChannel)"}, host.ErrInvalidID},
	}

	for i, tc := range testCases {
		err := tc.counterparty.ValidateBasic()
		if tc.expErr == nil {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
			require.ErrorIs(t, err, tc.expErr)
		}
	}
}

// TestIdentifiedChannelValidateBasic tests ValidateBasic for IdentifiedChannel.
func TestIdentifiedChannelValidateBasic(t *testing.T) {
	channel := types.NewChannel(types.TRYOPEN, types.ORDERED, types.Counterparty{"portidone", "channelidone"}, connHops, version)

	testCases := []struct {
		name              string
		identifiedChannel types.IdentifiedChannel
		expErr            error
	}{
		{
			"valid identified channel",
			types.NewIdentifiedChannel("portidone", "channelidone", channel),
			nil,
		},
		{
			"invalid portID",
			types.NewIdentifiedChannel("(InvalidPort)", "channelidone", channel),
			host.ErrInvalidID,
		},
		{
			"invalid channelID",
			types.NewIdentifiedChannel("portidone", "(InvalidChannel)", channel),
			host.ErrInvalidID,
		},
	}

	for _, tc := range testCases {
		err := tc.identifiedChannel.ValidateBasic()
		require.ErrorIs(t, err, tc.expErr)
	}
}
