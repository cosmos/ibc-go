package keeper

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

func (k Keeper) TrySendTx(ctx sdk.Context, accountOwner sdk.AccAddress, connectionId string, data interface{}) ([]byte, error) {
	portId := k.GeneratePortId(accountOwner.String(), connectionId)
	// Check for the active channel
	activeChannelId, found := k.GetActiveChannel(ctx, portId)
	if !found {
		return nil, types.ErrActiveChannelNotFound
	}

	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, portId, activeChannelId)
	if !found {
		return []byte{}, sdkerrors.Wrap(channeltypes.ErrChannelNotFound, activeChannelId)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	return k.createOutgoingPacket(ctx, portId, activeChannelId, destinationPort, destinationChannel, data)
}

func (k Keeper) createOutgoingPacket(
	ctx sdk.Context,
	sourcePort,
	sourceChannel,
	destinationPort,
	destinationChannel string,
	data interface{},
) ([]byte, error) {

	if data == nil {
		return []byte{}, types.ErrInvalidOutgoingData
	}

	var msgs []sdk.Msg

	switch data := data.(type) {
	case []sdk.Msg:
		msgs = data
	case sdk.Msg:
		msgs = []sdk.Msg{data}
	default:
		return []byte{}, types.ErrInvalidOutgoingData
	}

	txBytes, err := k.SerializeCosmosTx(k.cdc, msgs)
	if err != nil {
		return []byte{}, sdkerrors.Wrap(err, "invalid packet data or codec")
	}

	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
	if !ok {
		return []byte{}, sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	// get the next sequence
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return []byte{}, channeltypes.ErrSequenceSendNotFound
	}

	packetData := types.IBCAccountPacketData{
		Type: types.EXECUTE_TX,
		Data: txBytes,
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

	return k.ComputeVirtualTxHash(packetData.Data, packet.Sequence), k.channelKeeper.SendPacket(ctx, channelCap, packet)
}

func (k Keeper) DeserializeTx(_ sdk.Context, txBytes []byte) ([]sdk.Msg, error) {
	var txRaw types.IBCTxRaw

	err := k.cdc.Unmarshal(txBytes, &txRaw)
	if err != nil {
		return nil, err
	}

	var txBody types.IBCTxBody

	err = k.cdc.Unmarshal(txRaw.BodyBytes, &txBody)
	if err != nil {
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

	interchainAccountAddr, err := k.GetInterchainAccountAddress(ctx, portId)
	if err != nil {
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

// Compute the virtual tx hash that is used only internally.
func (k Keeper) ComputeVirtualTxHash(txBytes []byte, seq uint64) []byte {
	bz := make([]byte, 8)
	binary.LittleEndian.PutUint64(bz, seq)
	return tmhash.SumTruncated(append(txBytes, bz...))
}

func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	var data types.IBCAccountPacketData

	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal interchain account packet data: %s", err.Error())
	}

	switch data.Type {
	case types.EXECUTE_TX:
		msgs, err := k.DeserializeTx(ctx, data.Data)
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

func (k Keeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, data types.IBCAccountPacketData, ack channeltypes.Acknowledgement) error {
	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		if k.hook != nil {
			k.hook.OnTxFailed(ctx, packet.SourcePort, packet.SourceChannel, k.ComputeVirtualTxHash(data.Data, packet.Sequence), data.Data)
		}
		return nil
	case *channeltypes.Acknowledgement_Result:
		if k.hook != nil {
			k.hook.OnTxSucceeded(ctx, packet.SourcePort, packet.SourceChannel, k.ComputeVirtualTxHash(data.Data, packet.Sequence), data.Data)
		}
		return nil
	default:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		return nil
	}
}

func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, data types.IBCAccountPacketData) error {
	if k.hook != nil {
		k.hook.OnTxFailed(ctx, packet.SourcePort, packet.SourceChannel, k.ComputeVirtualTxHash(data.Data, packet.Sequence), data.Data)
	}

	return nil
}
