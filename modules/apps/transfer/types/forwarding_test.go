package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

var validHop = types.NewHop(types.PortID, ibctesting.FirstChannelID)

func TestForwarding_Validate(t *testing.T) {
	tests := []struct {
		name       string
		forwarding *types.Forwarding
		expError   error
	}{
		{
			"valid forwarding with no hops",
			types.NewForwarding(false),
			nil,
		},
		{
			"valid forwarding with hops",
			types.NewForwarding(false, validHop),
			nil,
		},
		{
			"valid forwarding with max hops",
			types.NewForwarding(false, generateHops(types.MaximumNumberOfForwardingHops)...),
			nil,
		},
		{
			"invalid forwarding with too many hops",
			types.NewForwarding(false, generateHops(types.MaximumNumberOfForwardingHops+1)...),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with too short hop port ID",
			types.NewForwarding(
				false,
				types.NewHop(invalidShortPort, ibctesting.FirstChannelID),
			),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with too long hop port ID",
			types.NewForwarding(
				false,
				types.NewHop(invalidLongPort, ibctesting.FirstChannelID),
			),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with non-alpha hop port ID",
			types.NewForwarding(
				false,
				types.NewHop(invalidPort, ibctesting.FirstChannelID),
			),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with too long hop channel ID",
			types.NewForwarding(
				false,
				types.NewHop(types.PortID, invalidLongChannel),
			),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with too short hop channel ID",
			types.NewForwarding(
				false,
				types.NewHop(types.PortID, invalidShortChannel),
			),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with non-alpha hop channel ID",
			types.NewForwarding(
				false,
				types.NewHop(types.PortID, invalidChannel),
			),
			types.ErrInvalidForwarding,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc

			err := tc.forwarding.Validate()

			expPass := tc.expError == nil
			if expPass {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expError)
			}
		})
	}
}

func TestForwardingPacketData_Validate(t *testing.T) {
	tests := []struct {
		name       string
		forwarding types.ForwardingPacketData
		expError   error
	}{
		{
			"valid forwarding with no hops",
			types.NewForwardingPacketData(""),
			nil,
		},
		{
			"valid forwarding with hops",
			types.NewForwardingPacketData("", validHop),
			nil,
		},
		{
			"valid forwarding with memo",
			types.NewForwardingPacketData(testMemo1, validHop, validHop),
			nil,
		},
		{
			"valid forwarding with max hops",
			types.NewForwardingPacketData("", generateHops(types.MaximumNumberOfForwardingHops)...),
			nil,
		},
		{
			"valid forwarding with max memo length",
			types.NewForwardingPacketData(ibctesting.GenerateString(types.MaximumMemoLength), validHop),
			nil,
		},
		{
			"invalid forwarding with too many hops",
			types.NewForwardingPacketData("", generateHops(types.MaximumNumberOfForwardingHops+1)...),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with too long memo",
			types.NewForwardingPacketData(ibctesting.GenerateString(types.MaximumMemoLength+1), validHop),
			types.ErrInvalidMemo,
		},
		{
			"invalid forwarding with empty hops and specified memo",
			types.NewForwardingPacketData("memo"),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with too short hop port ID",
			types.NewForwardingPacketData(
				"",
				types.NewHop(invalidShortPort, ibctesting.FirstChannelID),
			),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with too long hop port ID",
			types.NewForwardingPacketData(
				"",
				types.NewHop(invalidLongPort, ibctesting.FirstChannelID),
			),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with non-alpha hop port ID",
			types.NewForwardingPacketData(
				"",
				types.NewHop(invalidPort, ibctesting.FirstChannelID),
			),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with too long hop channel ID",
			types.NewForwardingPacketData(
				"",
				types.NewHop(types.PortID, invalidLongChannel),
			),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with too short hop channel ID",
			types.NewForwardingPacketData(
				"",
				types.NewHop(types.PortID, invalidShortChannel),
			),
			types.ErrInvalidForwarding,
		},
		{
			"invalid forwarding with non-alpha hop channel ID",
			types.NewForwardingPacketData(
				"",
				types.NewHop(types.PortID, invalidChannel),
			),
			types.ErrInvalidForwarding,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc

			err := tc.forwarding.Validate()

			expPass := tc.expError == nil
			if expPass {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expError)
			}
		})
	}
}

func TestValidateHop(t *testing.T) {
	tests := []struct {
		name     string
		hop      types.Hop
		expError error
	}{
		{
			"valid hop",
			validHop,
			nil,
		},
		{
			"invalid hop with too short port ID",
			types.NewHop(invalidShortPort, ibctesting.FirstChannelID),
			host.ErrInvalidID,
		},
		{
			"invalid hop with too long port ID",
			types.NewHop(invalidLongPort, ibctesting.FirstChannelID),
			host.ErrInvalidID,
		},
		{
			"invalid hop with non-alpha port ID",
			types.NewHop(invalidPort, ibctesting.FirstChannelID),
			host.ErrInvalidID,
		},
		{
			"invalid hop with too long channel ID",
			types.NewHop(types.PortID, invalidLongChannel),
			host.ErrInvalidID,
		},
		{
			"invalid hop with too short channel ID",
			types.NewHop(types.PortID, invalidShortChannel),
			host.ErrInvalidID,
		},
		{
			"invalid hop with non-alpha channel ID",
			types.NewHop(types.PortID, invalidChannel),
			host.ErrInvalidID,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc := tc

			err := tc.hop.Validate()

			expPass := tc.expError == nil
			if expPass {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expError)
			}
		})
	}
}

// generateHops generates a slice of n correctly initialized hops.
func generateHops(n int) []types.Hop {
	hops := make([]types.Hop, n)
	for i := 0; i < n; i++ {
		hops[i] = types.NewHop(types.PortID, ibctesting.FirstChannelID)
	}
	return hops
}
