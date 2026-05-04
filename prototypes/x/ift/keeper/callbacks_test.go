package keeper_test

import (
	"errors"
	"testing"

	"github.com/cosmos/ibc-go/prototypes/x/ift/types"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gmptypes "github.com/cosmos/ibc-go/v11/modules/apps/27-gmp/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
)

// TestCallbacks_IBCSendPacketCallback tests the IBCSendPacketCallback handler.
func TestCallbacks_IBCSendPacketCallback(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	// Should be a no-op and return nil
	err := wfapp.IFTKeeper.IBCSendPacketCallback(
		ctx,
		"", // sourceChannel
		"", // destChannel
		clienttypes.Height{},
		0,   // timeoutTimestamp
		nil, // data
		"",  // senderAddress
		"",  // receiverAddress
		"",  // memo
	)
	require.NoError(t, err)
}

// TestCallbacks_IBCOnAcknowledgementPacketCallback_Success tests successful acknowledgement handling.
func TestCallbacks_IBCOnAcknowledgementPacketCallback_Success(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)
	sequence := uint64(1)

	// Create token factory denom
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	// Setup pending transfer
	pending := types.PendingTransfer{
		Denom:    testDenom,
		ClientId: clientID,
		Sequence: sequence,
		Sender:   userAddrA,
		Amount:   math.NewInt(1000000),
	}
	err := wfapp.IFTKeeper.SetPendingTransfer(ctx, clientID, sequence, pending)
	require.NoError(t, err)

	// Create successful acknowledgement
	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	ackBytes, err := ack.Marshal()
	require.NoError(t, err)

	// Create packet
	packet := channeltypes.Packet{
		SourcePort:    gmptypes.PortID,
		SourceChannel: clientID,
		Sequence:      sequence,
	}

	moduleAddr := wfapp.IFTKeeper.GetModuleAddress()

	// Call the callback
	err = wfapp.IFTKeeper.IBCOnAcknowledgementPacketCallback(
		ctx,
		packet,
		ackBytes,
		moduleAddr,
		moduleAddr.String(),
		moduleAddr.String(),
		gmptypes.Version,
	)
	require.NoError(t, err)

	// Verify pending transfer was removed
	exists, err := wfapp.IFTKeeper.PendingTransferStore.Has(ctx, collections.Join(clientID, sequence))
	require.NoError(t, err)
	require.False(t, exists)
}

// TestCallbacks_IBCOnAcknowledgementPacketCallback_Error tests error acknowledgement handling with refund.
func TestCallbacks_IBCOnAcknowledgementPacketCallback_Error(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)
	sequence := uint64(1)

	// Create token factory denom
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	// Setup pending transfer
	pending := types.PendingTransfer{
		Denom:    testDenom,
		ClientId: clientID,
		Sequence: sequence,
		Sender:   userAddrA,
		Amount:   math.NewInt(1000000),
	}
	err := wfapp.IFTKeeper.SetPendingTransfer(ctx, clientID, sequence, pending)
	require.NoError(t, err)

	// Create v2 error acknowledgement (32-byte hash sentinel)
	ackBytes := channeltypesv2.ErrorAcknowledgement[:]

	// Create packet
	packet := channeltypes.Packet{
		SourcePort:    gmptypes.PortID,
		SourceChannel: clientID,
		Sequence:      sequence,
	}

	moduleAddr := wfapp.IFTKeeper.GetModuleAddress()

	// Get sender balance before
	senderAddr := sdk.MustAccAddressFromBech32(userAddrA)

	// Call the callback
	err = wfapp.IFTKeeper.IBCOnAcknowledgementPacketCallback(
		ctx,
		packet,
		ackBytes,
		moduleAddr,
		moduleAddr.String(),
		moduleAddr.String(),
		gmptypes.Version,
	)
	require.NoError(t, err)

	// Verify pending transfer was removed
	exists, err := wfapp.IFTKeeper.PendingTransferStore.Has(ctx, collections.Join(clientID, sequence))
	require.NoError(t, err)
	require.False(t, exists)

	// Verify tokens were refunded (minted back)
	balance := wfapp.BankKeeper.GetBalance(ctx, senderAddr, testDenom)
	require.True(t, balance.Amount.Equal(math.NewInt(1000000)))
}

// TestCallbacks_IBCOnAcknowledgementPacketCallback_V2AckFormat tests v2 ack format handling.
// In IBC v2, error is signaled ONLY by the 32-byte ErrorAcknowledgement sentinel hash.
// Any other bytes, including v1-style error ack protobuf, are treated as success.
func TestCallbacks_IBCOnAcknowledgementPacketCallback_V2AckFormat(t *testing.T) {
	cases := []struct {
		name         string
		ack          []byte
		expectRefund bool
	}{
		{
			name:         "success: raw bytes",
			ack:          []byte("success"),
			expectRefund: false,
		},
		{
			name:         "success: empty bytes",
			ack:          []byte{},
			expectRefund: false,
		},
		{
			name:         "error: v2 ErrorAcknowledgement sentinel triggers refund",
			ack:          channeltypesv2.ErrorAcknowledgement[:],
			expectRefund: true,
		},
		{
			name: "success: v1-style error ack is NOT treated as error in v2",
			ack: func() []byte {
				ack := channeltypes.NewErrorAcknowledgement(errors.New("some error"))
				bytes, _ := ack.Marshal()
				return bytes
			}(),
			expectRefund: false,
		},
		{
			name:         "success: partial match of error sentinel",
			ack:          channeltypesv2.ErrorAcknowledgement[:16],
			expectRefund: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wfapp, ctx := setupIntegrationApp(t)

			clientID := createIBCClient(t, ctx, wfapp)
			sequence := uint64(1)

			createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

			pending := types.PendingTransfer{
				Denom:    testDenom,
				ClientId: clientID,
				Sequence: sequence,
				Sender:   userAddrA,
				Amount:   math.NewInt(1000000),
			}
			err := wfapp.IFTKeeper.SetPendingTransfer(ctx, clientID, sequence, pending)
			require.NoError(t, err)

			packet := channeltypes.Packet{
				SourcePort:    gmptypes.PortID,
				SourceChannel: clientID,
				Sequence:      sequence,
			}

			moduleAddr := wfapp.IFTKeeper.GetModuleAddress()
			senderAddr := sdk.MustAccAddressFromBech32(userAddrA)

			err = wfapp.IFTKeeper.IBCOnAcknowledgementPacketCallback(
				ctx,
				packet,
				tc.ack,
				moduleAddr,
				moduleAddr.String(),
				moduleAddr.String(),
				gmptypes.Version,
			)
			require.NoError(t, err)

			exists, err := wfapp.IFTKeeper.PendingTransferStore.Has(ctx, collections.Join(clientID, sequence))
			require.NoError(t, err)
			require.False(t, exists, "pending transfer should be removed")

			balance := wfapp.BankKeeper.GetBalance(ctx, senderAddr, testDenom)
			if tc.expectRefund {
				require.True(t, balance.Amount.Equal(math.NewInt(1000000)), "tokens should be refunded")
			} else {
				require.True(t, balance.Amount.IsZero(), "tokens should NOT be refunded on success")
			}
		})
	}
}

// TestCallbacks_IBCOnAcknowledgementPacketCallback_NoPendingTransfer tests no-op when no pending transfer exists.
func TestCallbacks_IBCOnAcknowledgementPacketCallback_NoPendingTransfer(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)

	// Create successful acknowledgement
	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	ackBytes, err := ack.Marshal()
	require.NoError(t, err)

	// Create packet without corresponding pending transfer
	packet := channeltypes.Packet{
		SourcePort:    gmptypes.PortID,
		SourceChannel: clientID,
		Sequence:      999,
	}

	moduleAddr := wfapp.IFTKeeper.GetModuleAddress()

	// Should return nil (no-op for non-IFT packets)
	err = wfapp.IFTKeeper.IBCOnAcknowledgementPacketCallback(
		ctx,
		packet,
		ackBytes,
		moduleAddr,
		moduleAddr.String(),
		moduleAddr.String(),
		gmptypes.Version,
	)
	require.NoError(t, err)
}

// TestCallbacks_IBCOnAcknowledgementPacketCallback_NonIFTPacket tests that non-IFT packets are gracefully ignored.
// This is important because IFT is registered as the ContractKeeper for all GMP callbacks,
// but other applications may also use GMP. IFT must not interfere with their callbacks.
func TestCallbacks_IBCOnAcknowledgementPacketCallback_NonIFTPacket(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)

	// Create successful acknowledgement
	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	ackBytes, err := ack.Marshal()
	require.NoError(t, err)

	packet := channeltypes.Packet{
		SourcePort:    gmptypes.PortID,
		SourceChannel: clientID,
		Sequence:      1,
	}

	moduleAddr := wfapp.IFTKeeper.GetModuleAddress()

	cases := []struct {
		name            string
		contractAddress string
		packetSender    string
		version         string
		sourcePort      string
	}{
		{
			name:            "ignore: different contract address (other GMP app)",
			contractAddress: userAddrA,
			packetSender:    userAddrA,
			version:         gmptypes.Version,
			sourcePort:      gmptypes.PortID,
		},
		{
			name:            "ignore: different packet sender (other GMP app)",
			contractAddress: moduleAddr.String(),
			packetSender:    userAddrA,
			version:         gmptypes.Version,
			sourcePort:      gmptypes.PortID,
		},
		{
			name:            "ignore: different version",
			contractAddress: moduleAddr.String(),
			packetSender:    moduleAddr.String(),
			version:         "other-version",
			sourcePort:      gmptypes.PortID,
		},
		{
			name:            "ignore: different source port",
			contractAddress: moduleAddr.String(),
			packetSender:    moduleAddr.String(),
			version:         gmptypes.Version,
			sourcePort:      "other-port",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			packet.SourcePort = tc.sourcePort

			// Non-IFT packets should be gracefully ignored (return nil, no error)
			err := wfapp.IFTKeeper.IBCOnAcknowledgementPacketCallback(
				ctx,
				packet,
				ackBytes,
				moduleAddr,
				tc.contractAddress,
				tc.packetSender,
				tc.version,
			)
			require.NoError(t, err, "non-IFT packets should be gracefully ignored, not cause errors")
		})
	}
}

// TestCallbacks_IBCOnTimeoutPacketCallback tests packet timeout handling with refund.
func TestCallbacks_IBCOnTimeoutPacketCallback(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)
	sequence := uint64(1)

	// Create token factory denom
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	// Setup pending transfer
	pending := types.PendingTransfer{
		Denom:    testDenom,
		ClientId: clientID,
		Sequence: sequence,
		Sender:   userAddrA,
		Amount:   math.NewInt(500000),
	}
	err := wfapp.IFTKeeper.SetPendingTransfer(ctx, clientID, sequence, pending)
	require.NoError(t, err)

	// Create packet
	packet := channeltypes.Packet{
		SourcePort:    gmptypes.PortID,
		SourceChannel: clientID,
		Sequence:      sequence,
	}

	moduleAddr := wfapp.IFTKeeper.GetModuleAddress()
	senderAddr := sdk.MustAccAddressFromBech32(userAddrA)

	// Call the callback
	err = wfapp.IFTKeeper.IBCOnTimeoutPacketCallback(
		ctx,
		packet,
		moduleAddr,
		moduleAddr.String(),
		moduleAddr.String(),
		gmptypes.Version,
	)
	require.NoError(t, err)

	// Verify pending transfer was removed
	exists, err := wfapp.IFTKeeper.PendingTransferStore.Has(ctx, collections.Join(clientID, sequence))
	require.NoError(t, err)
	require.False(t, exists)

	// Verify tokens were refunded
	balance := wfapp.BankKeeper.GetBalance(ctx, senderAddr, testDenom)
	require.True(t, balance.Amount.Equal(math.NewInt(500000)))
}

// TestCallbacks_IBCOnTimeoutPacketCallback_NoPendingTransfer tests no-op when no pending transfer exists.
func TestCallbacks_IBCOnTimeoutPacketCallback_NoPendingTransfer(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)

	// Create packet without corresponding pending transfer
	packet := channeltypes.Packet{
		SourcePort:    gmptypes.PortID,
		SourceChannel: clientID,
		Sequence:      999,
	}

	moduleAddr := wfapp.IFTKeeper.GetModuleAddress()

	// Should return nil (no-op for non-IFT packets)
	err := wfapp.IFTKeeper.IBCOnTimeoutPacketCallback(
		ctx,
		packet,
		moduleAddr,
		moduleAddr.String(),
		moduleAddr.String(),
		gmptypes.Version,
	)
	require.NoError(t, err)
}

// TestCallbacks_IBCOnTimeoutPacketCallback_NonIFTPacket tests that non-IFT packets are gracefully ignored.
// This is important because IFT is registered as the ContractKeeper for all GMP callbacks,
// but other applications may also use GMP. IFT must not interfere with their callbacks.
func TestCallbacks_IBCOnTimeoutPacketCallback_NonIFTPacket(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)

	packet := channeltypes.Packet{
		SourcePort:    gmptypes.PortID,
		SourceChannel: clientID,
		Sequence:      1,
	}

	moduleAddr := wfapp.IFTKeeper.GetModuleAddress()

	cases := []struct {
		name            string
		contractAddress string
		packetSender    string
		version         string
		sourcePort      string
	}{
		{
			name:            "ignore: different contract address (other GMP app)",
			contractAddress: userAddrA,
			packetSender:    userAddrA,
			version:         gmptypes.Version,
			sourcePort:      gmptypes.PortID,
		},
		{
			name:            "ignore: different packet sender (other GMP app)",
			contractAddress: moduleAddr.String(),
			packetSender:    userAddrA,
			version:         gmptypes.Version,
			sourcePort:      gmptypes.PortID,
		},
		{
			name:            "ignore: different version",
			contractAddress: moduleAddr.String(),
			packetSender:    moduleAddr.String(),
			version:         "other-version",
			sourcePort:      gmptypes.PortID,
		},
		{
			name:            "ignore: different source port",
			contractAddress: moduleAddr.String(),
			packetSender:    moduleAddr.String(),
			version:         gmptypes.Version,
			sourcePort:      "other-port",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			packet.SourcePort = tc.sourcePort

			// Non-IFT packets should be gracefully ignored (return nil, no error)
			err := wfapp.IFTKeeper.IBCOnTimeoutPacketCallback(
				ctx,
				packet,
				moduleAddr,
				tc.contractAddress,
				tc.packetSender,
				tc.version,
			)
			require.NoError(t, err, "non-IFT packets should be gracefully ignored, not cause errors")
		})
	}
}

// TestCallbacks_IBCReceivePacketCallback tests the IBCReceivePacketCallback handler.
func TestCallbacks_IBCReceivePacketCallback(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	// Should be a no-op and return nil
	err := wfapp.IFTKeeper.IBCReceivePacketCallback(
		ctx,
		nil, // packet
		nil, // ack
		"",  // relayer
		"",  // contractAddress
	)
	require.NoError(t, err)
}

// TestCallbacks_RefundPendingTransfer tests the RefundPendingTransfer function.
func TestCallbacks_RefundPendingTransfer(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)
	sequence := uint64(42)

	// Create token factory denom
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	senderAddr := sdk.MustAccAddressFromBech32(userAddrA)

	// Setup pending transfer
	pending := types.PendingTransfer{
		Denom:    testDenom,
		ClientId: clientID,
		Sequence: sequence,
		Sender:   userAddrA,
		Amount:   math.NewInt(7500000),
	}
	err := wfapp.IFTKeeper.SetPendingTransfer(ctx, clientID, sequence, pending)
	require.NoError(t, err)

	// Get balance before refund (should be 0)
	balanceBefore := wfapp.BankKeeper.GetBalance(ctx, senderAddr, testDenom)
	require.True(t, balanceBefore.Amount.IsZero())

	// Refund
	err = wfapp.IFTKeeper.RefundPendingTransfer(ctx, testDenom, clientID, sequence)
	require.NoError(t, err)

	// Verify pending transfer was removed
	exists, err := wfapp.IFTKeeper.PendingTransferStore.Has(ctx, collections.Join(clientID, sequence))
	require.NoError(t, err)
	require.False(t, exists)

	// Verify tokens were minted back to sender
	balanceAfter := wfapp.BankKeeper.GetBalance(ctx, senderAddr, testDenom)
	require.True(t, balanceAfter.Amount.Equal(math.NewInt(7500000)))
}

// TestCallbacks_RefundPendingTransfer_NotFound tests RefundPendingTransfer when transfer not found.
func TestCallbacks_RefundPendingTransfer_NotFound(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)

	// Create token factory denom
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)

	// Try to refund non-existent pending transfer
	err := wfapp.IFTKeeper.RefundPendingTransfer(ctx, testDenom, clientID, 999)
	require.ErrorIs(t, err, types.ErrPendingTransferNotFound)
}

// TestCallbacks_MultiplePendingTransfers tests handling multiple pending transfers.
// Note: IBC sequences are unique per channel, so each pending transfer has a unique (clientId, sequence).
func TestCallbacks_MultiplePendingTransfers(t *testing.T) {
	wfapp, ctx := setupIntegrationApp(t)

	clientID := createIBCClient(t, ctx, wfapp)

	// Create token factory denoms
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom)
	createTokenFactoryDenom(t, ctx, wfapp, adminAddr, testDenom2)

	// Setup multiple pending transfers with unique sequences
	pending1 := types.PendingTransfer{
		Denom:    testDenom,
		ClientId: clientID,
		Sequence: 1,
		Sender:   userAddrA,
		Amount:   math.NewInt(100),
	}
	err := wfapp.IFTKeeper.SetPendingTransfer(ctx, clientID, 1, pending1)
	require.NoError(t, err)

	pending2 := types.PendingTransfer{
		Denom:    testDenom,
		ClientId: clientID,
		Sequence: 2,
		Sender:   userAddrB,
		Amount:   math.NewInt(200),
	}
	err = wfapp.IFTKeeper.SetPendingTransfer(ctx, clientID, 2, pending2)
	require.NoError(t, err)

	pending3 := types.PendingTransfer{
		Denom:    testDenom2,
		ClientId: clientID,
		Sequence: 3, // Unique sequence (IBC sequences are unique per channel)
		Sender:   userAddrA,
		Amount:   math.NewInt(300),
	}
	err = wfapp.IFTKeeper.SetPendingTransfer(ctx, clientID, 3, pending3)
	require.NoError(t, err)

	moduleAddr := wfapp.IFTKeeper.GetModuleAddress()

	// Timeout packet for sequence 1
	packet1 := channeltypes.Packet{
		SourcePort:    gmptypes.PortID,
		SourceChannel: clientID,
		Sequence:      1,
	}

	err = wfapp.IFTKeeper.IBCOnTimeoutPacketCallback(
		ctx,
		packet1,
		moduleAddr,
		moduleAddr.String(),
		moduleAddr.String(),
		gmptypes.Version,
	)
	require.NoError(t, err)

	// Verify pending1 was removed
	exists, err := wfapp.IFTKeeper.PendingTransferStore.Has(ctx, collections.Join(clientID, uint64(1)))
	require.NoError(t, err)
	require.False(t, exists)

	// pending2 and pending3 should still exist
	exists, err = wfapp.IFTKeeper.PendingTransferStore.Has(ctx, collections.Join(clientID, uint64(2)))
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = wfapp.IFTKeeper.PendingTransferStore.Has(ctx, collections.Join(clientID, uint64(3)))
	require.NoError(t, err)
	require.True(t, exists)

	// Verify userAddrA got refunded for testDenom
	senderAddrA := sdk.MustAccAddressFromBech32(userAddrA)
	balance := wfapp.BankKeeper.GetBalance(ctx, senderAddrA, testDenom)
	require.True(t, balance.Amount.Equal(math.NewInt(100)))
}
