package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v2/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
)

// TODO: implement middleware functionality, this will allow us to use capabilities to
// manage helper module access to owner addresses they do not have capabilities for
func (k Keeper) TrySendTx(ctx sdk.Context, portID string, data interface{}, memo string) (uint64, error) {
	// Check for the active channel
	activeChannelId, found := k.GetActiveChannel(ctx, portID)
	if !found {
		return 0, types.ErrActiveChannelNotFound
	}

	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, portID, activeChannelId)
	if !found {
		return 0, sdkerrors.Wrap(channeltypes.ErrChannelNotFound, activeChannelId)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	return k.createOutgoingPacket(ctx, portID, activeChannelId, destinationPort, destinationChannel, data, memo)
}

func (k Keeper) createOutgoingPacket(
	ctx sdk.Context,
	sourcePort,
	sourceChannel,
	destinationPort,
	destinationChannel string,
	data interface{},
	memo string,
) (uint64, error) {
	if data == nil {
		return 0, types.ErrInvalidOutgoingData
	}

	var (
		txBytes []byte
		err     error
	)

	switch data := data.(type) {
	case []sdk.Msg:
		txBytes, err = k.SerializeCosmosTx(k.cdc, data)
	default:
		return 0, sdkerrors.Wrapf(types.ErrInvalidOutgoingData, "message type %T is not supported", data)
	}

	if err != nil {
		return 0, sdkerrors.Wrap(err, "serialization of transaction data failed")
	}

	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
	if !ok {
		return 0, sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	// get the next sequence
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, channeltypes.ErrSequenceSendNotFound
	}

	packetData := types.InterchainAccountPacketData{
		Type: types.EXECUTE_TX,
		Data: txBytes,
		Memo: memo,
	}

	// timeoutTimestamp is set to be a max number here so that we never recieve a timeout
	// ics-27-1 uses ordered channels which can close upon recieving a timeout, which is an undesired effect
	const timeoutTimestamp = ^uint64(0) >> 1 // Shift the unsigned bit to satisfy hermes relayer timestamp conversion

	packet := channeltypes.NewPacket(
		packetData.GetBytes(),
		sequence,
		sourcePort,
		sourceChannel,
		destinationPort,
		destinationChannel,
		clienttypes.ZeroHeight(),
		timeoutTimestamp,
	)

	if err := k.channelKeeper.SendPacket(ctx, channelCap, packet); err != nil {
		return 0, err
	}

	return packet.Sequence, nil
}

// DeserializeCosmosTx unmarshals and unpacks a slice of transaction bytes
// into a slice of sdk.Msg's.
func (k Keeper) DeserializeCosmosTx(_ sdk.Context, txBytes []byte) ([]sdk.Msg, error) {
	var txBody types.IBCTxBody

	if err := k.cdc.Unmarshal(txBytes, &txBody); err != nil {
		return nil, err
	}

	anys := txBody.Messages
	res := make([]sdk.Msg, len(anys))
	for i, any := range anys {
		var msg sdk.Msg
		err := k.cdc.UnpackAny(any, &msg)
		if err != nil {
			return nil, err
		}
		res[i] = msg
	}

	return res, nil
}

func (k Keeper) AuthenticateTx(ctx sdk.Context, msgs []sdk.Msg, portId string) error {
	seen := map[string]bool{}
	var signers []sdk.AccAddress
	for _, msg := range msgs {
		for _, addr := range msg.GetSigners() {
			if !seen[addr.String()] {
				signers = append(signers, addr)
				seen[addr.String()] = true
			}
		}
	}

	interchainAccountAddr, found := k.GetInterchainAccountAddress(ctx, portId)
	if !found {
		return sdkerrors.ErrUnauthorized
	}

	for _, signer := range signers {
		if interchainAccountAddr != signer.String() {
			return sdkerrors.ErrUnauthorized
		}
	}

	return nil
}

func (k Keeper) executeTx(ctx sdk.Context, sourcePort, destPort, destChannel string, msgs []sdk.Msg) error {
	err := k.AuthenticateTx(ctx, msgs, sourcePort)
	if err != nil {
		return err
	}

	for _, msg := range msgs {
		err := msg.ValidateBasic()
		if err != nil {
			return err
		}
	}

	cacheContext, writeFn := ctx.CacheContext()
	for _, msg := range msgs {
		_, msgErr := k.executeMsg(cacheContext, msg)
		if msgErr != nil {
			err = msgErr
			break
		}
	}

	if err != nil {
		return err
	}

	// Write the state transitions if all handlers succeed.
	writeFn()

	return nil
}

// It tries to get the handler from router. And, if router exites, it will perform message.
func (k Keeper) executeMsg(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
	handler := k.msgRouter.Handler(msg)
	if handler == nil {
		return nil, types.ErrInvalidRoute
	}

	return handler(ctx, msg)
}

func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	var data types.InterchainAccountPacketData

	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return sdkerrors.Wrapf(types.ErrUnknownPacketData, "cannot unmarshal ICS-27 interchain account packet data")
	}

	switch data.Type {
	case types.EXECUTE_TX:
		msgs, err := k.DeserializeCosmosTx(ctx, data.Data)
		if err != nil {
			return err
		}

		err = k.executeTx(ctx, packet.SourcePort, packet.DestinationPort, packet.DestinationChannel, msgs)
		if err != nil {
			return err
		}

		return nil
	default:
		return types.ErrUnknownPacketData
	}
}

func (k Keeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, data types.InterchainAccountPacketData, ack channeltypes.Acknowledgement) error {
	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		if k.hook != nil {
			k.hook.OnTxFailed(ctx, packet.SourcePort, packet.SourceChannel, packet.Data, data.Data)
		}
		return nil
	case *channeltypes.Acknowledgement_Result:
		if k.hook != nil {
			k.hook.OnTxSucceeded(ctx, packet.SourcePort, packet.SourceChannel, packet.Data, data.Data)
		}
		return nil
	default:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		return nil
	}
}

// OnTimeoutPacket removes the active channel associated with the provided packet, the underlying channel end is closed
// due to the semantics of ORDERED channels
func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	k.DeleteActiveChannel(ctx, packet.SourcePort)

	return nil
}
