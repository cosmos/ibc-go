package v2

import (
	"context"
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/internal"
	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/modules/core/api"
)

var _ api.IBCModule = (*IBCMiddleware)(nil)

// IBCMiddleware implements the IBC v2 middleware interface
// with the underlying application.
type IBCMiddleware struct {
	app             types.CallbacksCompatibleModuleV2
	writeAckWrapper api.WriteAcknowledgementWrapper

	contractKeeper types.ContractKeeper
	chanKeeperV2   types.ChannelKeeperV2

	// maxCallbackGas defines the maximum amount of gas that a callback actor can ask the
	// relayer to pay for. If a callback fails due to insufficient gas, the entire tx
	// is reverted if the relayer hadn't provided the minimum(userDefinedGas, maxCallbackGas).
	// If the actor hasn't defined a gas limit, then it is assumed to be the maxCallbackGas.
	maxCallbackGas uint64
}

// NewIBCMiddleware creates a new IBCMiddleware instance given the keeper and underlying application.
// The underlying application must implement the required callback interfaces.
func NewIBCMiddleware(
	app api.IBCModule, writeAckWrapper api.WriteAcknowledgementWrapper,
	contractKeeper types.ContractKeeper, chanKeeperV2 types.ChannelKeeperV2, maxCallbackGas uint64,
) IBCMiddleware {
	packetDataUnmarshalerApp, ok := app.(types.CallbacksCompatibleModuleV2)
	if !ok {
		panic(fmt.Errorf("underlying application does not implement %T", (*types.CallbacksCompatibleModule)(nil)))
	}

	if contractKeeper == nil {
		panic(errors.New("contract keeper cannot be nil"))
	}

	if writeAckWrapper == nil {
		panic(errors.New("write acknowledgement wrapper cannot be nil"))
	}

	if chanKeeperV2 == nil {
		panic(errors.New("channel keeper v2 cannot be nil"))
	}

	if maxCallbackGas == 0 {
		panic(errors.New("maxCallbackGas cannot be zero"))
	}

	return IBCMiddleware{
		app:             packetDataUnmarshalerApp,
		writeAckWrapper: writeAckWrapper,
		contractKeeper:  contractKeeper,
		chanKeeperV2:    chanKeeperV2,
		maxCallbackGas:  maxCallbackGas,
	}
}

// WithWriteAckWrapper sets the WriteAcknowledgementWrapper for the middleware.
func (im *IBCMiddleware) WithWriteAckWrapper(writeAckWrapper api.WriteAcknowledgementWrapper) {
	im.writeAckWrapper = writeAckWrapper
}

// GetWriteAckWrapper returns the WriteAckWrapper
func (im *IBCMiddleware) GetWriteAckWrapper() api.WriteAcknowledgementWrapper {
	return im.writeAckWrapper
}

// OnSendPacket implements source callbacks for sending packets.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback returns an error, panics, or runs out of gas, then
// the packet send is rejected.
func (im IBCMiddleware) OnSendPacket(
	ctx context.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	signer sdk.AccAddress,
) error {
	err := im.app.OnSendPacket(ctx, sourceClient, destinationClient, sequence, payload, signer)
	if err != nil {
		return err
	}

	packetData, err := im.app.UnmarshalPacketData(payload)
	// OnSendPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cbData, err := types.GetCallbackData(
		packetData, payload.GetVersion(), payload.GetSourcePort(),
		sdkCtx.GasMeter().GasRemaining(), im.maxCallbackGas, types.SourceCallbackKey,
	)
	// OnSendPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		return im.contractKeeper.IBCSendPacketCallback(
			cachedCtx, payload.SourcePort, sourceClient, clienttypes.Height{}, 0, payload.Value, cbData.CallbackAddress, cbData.SenderAddress, payload.Version,
		)
	}

	err = internal.ProcessCallback(sdkCtx, types.CallbackTypeSendPacket, cbData, callbackExecutor)
	// contract keeper is allowed to reject the packet send.
	if err != nil {
		return err
	}

	types.EmitCallbackEvent(sdkCtx, payload.SourcePort, sourceClient, sequence, types.CallbackTypeSendPacket, cbData, nil)
	return nil
}

// OnRecvPacket implements the ReceivePacket destination callbacks for the ibc-callbacks middleware during
// synchronous packet acknowledgement.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit then the state changes are
// reverted via a panic.
func (im IBCMiddleware) OnRecvPacket(
	ctx context.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) channeltypesv2.RecvPacketResult {
	recvResult := im.app.OnRecvPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
	// if ack is nil (asynchronous acknowledgements), then the callback will be handled in WriteAcknowledgement
	// if ack is not successful, all state changes are reverted. If a packet cannot be received, then there is
	// no need to execute a callback on the receiving chain.
	if recvResult.Status == channeltypesv2.PacketStatus_Async || recvResult.Status == channeltypesv2.PacketStatus_Failure {
		return recvResult
	}

	packetData, err := im.app.UnmarshalPacketData(payload)
	// OnRecvPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return recvResult
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cbData, err := types.GetCallbackData(
		packetData, payload.GetVersion(), payload.GetDestinationPort(),
		sdkCtx.GasMeter().GasRemaining(), im.maxCallbackGas, types.DestinationCallbackKey,
	)
	// OnRecvPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return recvResult
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		// reconstruct a channel v1 packet from the v2 packet
		// in order to preserve the same interface for the contract keeper
		packetv1 := channeltypes.Packet{
			Sequence:           sequence,
			SourcePort:         payload.SourcePort,
			SourceChannel:      sourceClient,
			DestinationPort:    payload.DestinationPort,
			DestinationChannel: destinationClient,
			Data:               payload.Value,
			TimeoutHeight:      clienttypes.Height{},
			TimeoutTimestamp:   0,
		}
		// wrap the individual acknowledgement into the channeltypesv2.Acknowledgement since it implements the exported.Acknowledgement interface
		// since we return early on failure, we are guaranteed that the ack is a successful acknowledgement
		ack := channeltypesv2.NewAcknowledgement(recvResult.Acknowledgement)
		return im.contractKeeper.IBCReceivePacketCallback(cachedCtx, packetv1, ack, cbData.CallbackAddress, payload.Version)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = internal.ProcessCallback(sdkCtx, types.CallbackTypeReceivePacket, cbData, callbackExecutor)
	types.EmitCallbackEvent(
		sdkCtx, payload.DestinationPort, destinationClient, sequence,
		types.CallbackTypeReceivePacket, cbData, err,
	)

	return recvResult
}

// OnAcknowledgementPacket implements source callbacks for acknowledgement packets.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit then the state changes are
// reverted via a panic.
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx context.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	acknowledgement []byte,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	// we first call the underlying app to handle the acknowledgement
	err := im.app.OnAcknowledgementPacket(ctx, sourceClient, destinationClient, sequence, acknowledgement, payload, relayer)
	if err != nil {
		return err
	}

	packetData, err := im.app.UnmarshalPacketData(payload)
	// OnAcknowledgementPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cbData, err := types.GetCallbackData(
		packetData, payload.GetVersion(), payload.GetSourcePort(),
		sdkCtx.GasMeter().GasRemaining(), im.maxCallbackGas, types.SourceCallbackKey,
	)
	// OnAcknowledgementPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		// reconstruct a channel v1 packet from the v2 packet
		// in order to preserve the same interface for the contract keeper
		packetv1 := channeltypes.Packet{
			Sequence:           sequence,
			SourcePort:         payload.SourcePort,
			SourceChannel:      sourceClient,
			DestinationPort:    payload.DestinationPort,
			DestinationChannel: destinationClient,
			Data:               payload.Value,
			TimeoutHeight:      clienttypes.Height{},
			TimeoutTimestamp:   0,
		}
		// NOTE: The callback is receiving the acknowledgement that the application received for its particular payload.
		// In the case of a successful acknowledgement, this will be the acknowledgement sent by the counterparty application for the given payload
		// In the case of an error acknowledgement, this will be the sentinel error acknowledgement bytes defined by IBC v2 protocol.
		// Thus, the contract must be aware that the sentinel error acknowledgement signals a failed receive
		// and the contract must handle this error case and the corresponding success case (ie ack != ErrorAcknowledgement) accordingly.
		return im.contractKeeper.IBCOnAcknowledgementPacketCallback(
			cachedCtx, packetv1, acknowledgement, relayer, cbData.CallbackAddress, cbData.SenderAddress, payload.Version,
		)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = internal.ProcessCallback(sdkCtx, types.CallbackTypeAcknowledgementPacket, cbData, callbackExecutor)
	types.EmitCallbackEvent(
		sdkCtx, payload.SourcePort, sourceClient, sequence,
		types.CallbackTypeAcknowledgementPacket, cbData, err,
	)

	return nil
}

// OnTimeoutPacket implements timeout source callbacks for the ibc-callbacks middleware.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit then the state changes are
// reverted via a panic.
// OnTimeoutPacket is executed when a packet has timed out on the receiving chain.
func (im IBCMiddleware) OnTimeoutPacket(
	ctx context.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	err := im.app.OnTimeoutPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
	if err != nil {
		return err
	}

	packetData, err := im.app.UnmarshalPacketData(payload)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cbData, err := types.GetCallbackData(
		packetData, payload.GetVersion(), payload.GetSourcePort(),
		sdkCtx.GasMeter().GasRemaining(), im.maxCallbackGas, types.SourceCallbackKey,
	)
	// OnTimeoutPacket is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	callbackExecutor := func(cachedCtx sdk.Context) error {
		// reconstruct a channel v1 packet from the v2 packet
		// in order to preserve the same interface for the contract keeper
		packetv1 := channeltypes.Packet{
			Sequence:           sequence,
			SourcePort:         payload.SourcePort,
			SourceChannel:      sourceClient,
			DestinationPort:    payload.DestinationPort,
			DestinationChannel: destinationClient,
			Data:               payload.Value,
			TimeoutHeight:      clienttypes.Height{},
			TimeoutTimestamp:   0,
		}
		return im.contractKeeper.IBCOnTimeoutPacketCallback(
			cachedCtx, packetv1, relayer, cbData.CallbackAddress, cbData.SenderAddress, payload.Version,
		)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = internal.ProcessCallback(sdkCtx, types.CallbackTypeTimeoutPacket, cbData, callbackExecutor)
	types.EmitCallbackEvent(
		sdkCtx, payload.SourcePort, sourceClient, sequence,
		types.CallbackTypeTimeoutPacket, cbData, err,
	)

	return nil
}

// WriteAcknowledgement implements the ReceivePacket destination callbacks for the ibc-callbacks middleware
// during asynchronous packet acknowledgement.
// It defers to the underlying application and then calls the contract callback.
// If the contract callback runs out of gas and may be retried with a higher gas limit then the state changes are
// reverted via a panic.
func (im IBCMiddleware) WriteAcknowledgement(
	ctx context.Context,
	clientID string,
	sequence uint64,
	ack channeltypesv2.Acknowledgement,
) error {
	packet, found := im.chanKeeperV2.GetAsyncPacket(ctx, clientID, sequence)
	if !found {
		return errorsmod.Wrapf(channeltypesv2.ErrInvalidAcknowledgement, "async packet not found for clientID (%s) and sequence (%d)", clientID, sequence)
	}

	err := im.writeAckWrapper.WriteAcknowledgement(ctx, clientID, sequence, ack)
	if err != nil {
		return err
	}

	// NOTE: use first payload as the payload that is being handled by callbacks middleware
	// must reconsider if multipacket data gets supported with async packets
	// TRACKING ISSUE: https://github.com/cosmos/ibc-go/issues/7950
	if len(packet.Payloads) != 1 {
		return errorsmod.Wrapf(channeltypesv2.ErrInvalidAcknowledgement, "async packet has multiple payloads")
	}
	payload := packet.Payloads[0]

	packetData, err := im.app.UnmarshalPacketData(payload)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cbData, err := types.GetCallbackData(
		packetData, payload.GetVersion(), payload.GetDestinationPort(),
		sdkCtx.GasMeter().GasRemaining(), im.maxCallbackGas, types.DestinationCallbackKey,
	)
	// WriteAcknowledgement is not blocked if the packet does not opt-in to callbacks
	if err != nil {
		return nil
	}

	recvResult := channeltypesv2.RecvPacketResult{
		Status:          channeltypesv2.PacketStatus_Success,
		Acknowledgement: ack.AppAcknowledgements[0],
	}
	callbackExecutor := func(cachedCtx sdk.Context) error {
		// reconstruct a channel v1 packet from the v2 packet
		// in order to preserve the same interface for the contract keeper
		packetv1 := channeltypes.Packet{
			Sequence:           sequence,
			SourcePort:         payload.SourcePort,
			SourceChannel:      packet.SourceClient,
			DestinationPort:    payload.DestinationPort,
			DestinationChannel: packet.DestinationClient,
			Data:               payload.Value,
			TimeoutHeight:      clienttypes.Height{},
			TimeoutTimestamp:   0,
		}
		// wrap the individual acknowledgement into the channeltypesv2.Acknowledgement since it implements the exported.Acknowledgement interface
		var ack channeltypesv2.Acknowledgement
		if recvResult.Status == channeltypesv2.PacketStatus_Failure {
			ack = channeltypesv2.NewAcknowledgement(channeltypesv2.ErrorAcknowledgement[:])
		} else {
			ack = channeltypesv2.NewAcknowledgement(recvResult.Acknowledgement)
		}
		return im.contractKeeper.IBCReceivePacketCallback(
			cachedCtx, packetv1, ack, cbData.CallbackAddress, payload.Version,
		)
	}

	// callback execution errors are not allowed to block the packet lifecycle, they are only used in event emissions
	err = internal.ProcessCallback(sdkCtx, types.CallbackTypeReceivePacket, cbData, callbackExecutor)
	types.EmitCallbackEvent(
		sdkCtx, payload.DestinationPort, clientID, sequence,
		types.CallbackTypeReceivePacket, cbData, err,
	)

	return nil
}
