package keeper

import (
	"bytes"
	"strconv"

	"github.com/cosmos/ibc-go/prototypes/x/ift/types"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	gmptypes "github.com/cosmos/ibc-go/v11/modules/apps/27-gmp/types"
	callbacktypes "github.com/cosmos/ibc-go/v11/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
)

// Ensure IFT Keeper implements the ContractKeeper interface for IBC callbacks
var _ callbacktypes.ContractKeeper = (*Keeper)(nil)

// IBCSendPacketCallback is called when a packet is sent.
// This is a no-op for IFT as no special handling is needed.
func (Keeper) IBCSendPacketCallback(
	_ sdk.Context,
	_ string,
	_ string,
	_ clienttypes.Height,
	_ uint64,
	_ []byte,
	_, _ string,
	_ string,
) error {
	return nil
}

// IBCOnAcknowledgementPacketCallback is called when a packet acknowledgement is received.
func (k Keeper) IBCOnAcknowledgementPacketCallback(
	cachedCtx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	_ sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
	version string,
) error {
	k.Logger(cachedCtx).Debug("IBCOnAcknowledgementPacketCallback called",
		"contractAddress", contractAddress,
		"packetSenderAddress", packetSenderAddress,
		"version", version,
		"sourcePort", packet.SourcePort,
		"sourceChannel", packet.SourceChannel,
		"sequence", packet.Sequence)

	// The callbacks middleware is registered on the GMP IBC module, so all
	// callbacks here originate from GMP packets. Since multiple applications
	// could use GMP, we check if this specific callback belongs to an IFT
	// transfer. If not, return nil to pass control to other callback handlers.
	if err := k.checkIsIFTCallback(cachedCtx, contractAddress, packetSenderAddress, version, packet.SourcePort); err != nil {
		k.Logger(cachedCtx).Debug("IFT ack callback: skipping non-IFT packet",
			"reason", err.Error(),
			"contractAddress", contractAddress,
			"packetSenderAddress", packetSenderAddress)
		return nil
	}

	// In IBC v2, the acknowledgement is the raw app ack bytes.
	// Error is signaled by the sentinel ErrorAcknowledgement hash.
	isErrorAck := bytes.Equal(acknowledgement, channeltypesv2.ErrorAcknowledgement[:])

	k.Logger(cachedCtx).Debug("IFT ack callback processing",
		"isErrorAck", isErrorAck,
		"ackLen", len(acknowledgement))

	// Find the pending transfer for this packet using O(1) index lookup
	pending, found, err := k.GetPendingTransferByClientSequence(cachedCtx, packet.SourceChannel, packet.Sequence)
	if err != nil {
		return err
	}

	if !found {
		k.Logger(cachedCtx).Debug("IFT ack callback: skipping non-IFT packet",
			"clientId", packet.SourceChannel,
			"sequence", packet.Sequence)
		return nil
	}

	k.Logger(cachedCtx).Debug("IFT ack callback: found pending transfer",
		"denom", pending.Denom,
		"clientId", pending.ClientId,
		"sequence", pending.Sequence)

	if isErrorAck {
		// Acknowledgement indicates error, refund the sender
		k.Logger(cachedCtx).Info("IFT ack callback: error ack, refunding")
		return k.RefundPendingTransfer(cachedCtx, pending.Denom, pending.ClientId, pending.Sequence)
	}

	// Success - remove pending transfer
	if err := k.RemovePendingTransfer(cachedCtx, pending.ClientId, pending.Sequence); err != nil {
		return err
	}

	cachedCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeIFTTransferCompleted,
			sdk.NewAttribute(types.AttributeKeyDenom, pending.Denom),
			sdk.NewAttribute(types.AttributeKeyClientID, pending.ClientId),
			sdk.NewAttribute(types.AttributeKeySequence, strconv.FormatUint(pending.Sequence, 10)),
			sdk.NewAttribute(types.AttributeKeySender, pending.Sender),
			sdk.NewAttribute(types.AttributeKeyAmount, pending.Amount.String()),
		),
	)

	k.Logger(cachedCtx).Info("IFT transfer completed",
		"denom", pending.Denom,
		"client_id", pending.ClientId,
		"sequence", pending.Sequence,
		"sender", pending.Sender,
		"amount", pending.Amount.String())

	return nil
}

// IBCOnTimeoutPacketCallback is called when a packet times out.
func (k Keeper) IBCOnTimeoutPacketCallback(
	cachedCtx sdk.Context,
	packet channeltypes.Packet,
	_ sdk.AccAddress,
	contractAddress,
	packetSenderAddress string,
	version string,
) error {
	k.Logger(cachedCtx).Debug("IBCOnTimeoutPacketCallback called",
		"contractAddress", contractAddress,
		"packetSenderAddress", packetSenderAddress,
		"version", version,
		"sourcePort", packet.SourcePort,
		"sourceChannel", packet.SourceChannel,
		"sequence", packet.Sequence)

	// The callbacks middleware is registered on the GMP IBC module, so all
	// callbacks here originate from GMP packets. Since multiple applications
	// could use GMP, we check if this specific callback belongs to an IFT
	// transfer. If not, return nil to pass control to other callback handlers.
	if err := k.checkIsIFTCallback(cachedCtx, contractAddress, packetSenderAddress, version, packet.SourcePort); err != nil {
		k.Logger(cachedCtx).Debug("IFT timeout callback: skipping non-IFT packet",
			"reason", err.Error(),
			"contractAddress", contractAddress,
			"packetSenderAddress", packetSenderAddress)
		return nil
	}

	// Find the pending transfer for this packet using O(1) index lookup
	pending, found, err := k.GetPendingTransferByClientSequence(cachedCtx, packet.SourceChannel, packet.Sequence)
	if err != nil {
		return err
	}

	if !found {
		k.Logger(cachedCtx).Debug("IFT timeout callback: skipping non-IFT packet",
			"clientId", packet.SourceChannel,
			"sequence", packet.Sequence)
		return nil
	}

	return k.RefundPendingTransfer(cachedCtx, pending.Denom, pending.ClientId, pending.Sequence)
}

// IBCReceivePacketCallback is not supported for IFT.
// Minting is handled via the IFTMint message handler.
func (Keeper) IBCReceivePacketCallback(
	_ sdk.Context,
	_ ibcexported.PacketI,
	_ ibcexported.Acknowledgement,
	_ string,
	_ string,
) error {
	return nil
}

// checkIsIFTCallback checks if the callback belongs to IFT by matching
// the contract address, packet sender, version, and source port.
// Returns nil if this is an IFT callback, or an error explaining why not.
func (k Keeper) checkIsIFTCallback(_ sdk.Context, contractAddress, packetSenderAddress, version, sourcePort string) error {
	moduleAddr := k.GetModuleAddress().String()

	if contractAddress != moduleAddr {
		return errorsmod.Wrapf(types.ErrCallbackValidationFailed, "expected contract address %s, got %s", moduleAddr, contractAddress)
	}

	if packetSenderAddress != moduleAddr {
		return errorsmod.Wrapf(types.ErrCallbackValidationFailed, "expected packet sender %s, got %s", moduleAddr, packetSenderAddress)
	}

	if version != gmptypes.Version {
		return errorsmod.Wrapf(types.ErrCallbackValidationFailed, "expected version %s, got %s", gmptypes.Version, version)
	}

	if sourcePort != gmptypes.PortID {
		return errorsmod.Wrapf(types.ErrCallbackValidationFailed, "expected source port %s, got %s", gmptypes.PortID, sourcePort)
	}

	return nil
}

// RefundPendingTransfer refunds a pending transfer and removes it from storage
func (k Keeper) RefundPendingTransfer(ctx sdk.Context, denom, clientID string, sequence uint64) error {
	pending, err := k.PendingTransferStore.Get(ctx, collections.Join(clientID, sequence))
	if err != nil {
		return errorsmod.Wrapf(types.ErrPendingTransferNotFound, "client %s, sequence %d", clientID, sequence)
	}

	sender, err := k.addressCodec.StringToBytes(pending.Sender)
	if err != nil {
		return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid sender address: %s", err)
	}

	// Remove pending transfer first (checks-effects-interactions pattern)
	// In Cosmos SDK, if MintTo fails, the entire transaction reverts including this removal
	if err := k.RemovePendingTransfer(ctx, clientID, sequence); err != nil {
		return err
	}

	// Mint tokens back to sender
	if err := k.tokenFactoryKeeper.MintTo(ctx, pending.Denom, pending.Amount, sender); err != nil {
		return errorsmod.Wrapf(types.ErrMintFailed, "failed to refund tokens: %s", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeIFTTransferRefunded,
			sdk.NewAttribute(types.AttributeKeyDenom, pending.Denom),
			sdk.NewAttribute(types.AttributeKeyClientID, pending.ClientId),
			sdk.NewAttribute(types.AttributeKeySequence, strconv.FormatUint(pending.Sequence, 10)),
			sdk.NewAttribute(types.AttributeKeySender, pending.Sender),
			sdk.NewAttribute(types.AttributeKeyAmount, pending.Amount.String()),
		),
	)

	k.Logger(ctx).Info("IFT transfer refunded",
		"denom", pending.Denom,
		"client_id", pending.ClientId,
		"sequence", pending.Sequence,
		"sender", pending.Sender,
		"amount", pending.Amount.String())

	return nil
}
