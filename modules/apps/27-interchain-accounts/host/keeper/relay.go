package keeper

import (
	"context"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

// OnRecvPacket handles a given interchain accounts packet on a destination host chain.
// If the transaction is successfully executed, the transaction response bytes will be returned.
func (k Keeper) OnRecvPacket(ctx context.Context, packet channeltypes.Packet) ([]byte, error) {
	var data icatypes.InterchainAccountPacketData
	err := data.UnmarshalJSON(packet.GetData())
	if err != nil {
		// UnmarshalJSON errors are indeterminate and therefore are not wrapped and included in failed acks
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot unmarshal ICS-27 interchain account packet data")
	}

	metadata, err := k.getAppMetadata(ctx, packet.DestinationPort, packet.DestinationChannel)
	if err != nil {
		return nil, err
	}

	switch data.Type {
	case icatypes.EXECUTE_TX:
		msgs, err := icatypes.DeserializeCosmosTx(k.cdc, data.Data, metadata.Encoding)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed to deserialize interchain account transaction")
		}

		txResponse, err := k.executeTx(ctx, packet.SourcePort, packet.DestinationPort, packet.DestinationChannel, msgs)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "failed to execute interchain account transaction")
		}
		return txResponse, nil
	default:
		return nil, icatypes.ErrUnknownDataType
	}
}

// executeTx attempts to execute the provided transaction. It begins by authenticating the transaction signer.
// If authentication succeeds, it does basic validation of the messages before attempting to deliver each message
// into state. The state changes will only be committed if all messages in the transaction succeed. Thus the
// execution of the transaction is atomic, all state changes are reverted if a single message fails.
func (k Keeper) executeTx(ctx context.Context, sourcePort, destPort, destChannel string, msgs []sdk.Msg) ([]byte, error) {
	channel, found := k.channelKeeper.GetChannel(ctx, destPort, destChannel)
	if !found {
		return nil, channeltypes.ErrChannelNotFound
	}

	if err := k.authenticateTx(ctx, msgs, channel.ConnectionHops[0], sourcePort); err != nil {
		return nil, err
	}

	txMsgData := &sdk.TxMsgData{
		MsgResponses: make([]*codectypes.Any, len(msgs)),
	}

	for i, msg := range msgs {
		if m, ok := msg.(sdk.HasValidateBasic); ok {
			if err := m.ValidateBasic(); err != nil {
				return nil, err
			}
		}

		if err := k.BranchService.Execute(ctx, func(ctx context.Context) error {
			protoAny, err := k.executeMsg(ctx, msg)
			if err != nil {
				return err
			}

			txMsgData.MsgResponses[i] = protoAny
			return nil
		}); err != nil {
			return nil, err
		}
	}

	txResponse, err := proto.Marshal(txMsgData)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to marshal tx data")
	}

	return txResponse, nil
}

// authenticateTx ensures the provided msgs contain the correct interchain account signer address retrieved
// from state using the provided controller port identifier
func (k Keeper) authenticateTx(ctx context.Context, msgs []sdk.Msg, connectionID, portID string) error {
	interchainAccountAddr, found := k.GetInterchainAccountAddress(ctx, connectionID, portID)
	if !found {
		return errorsmod.Wrapf(icatypes.ErrInterchainAccountNotFound, "failed to retrieve interchain account on port %s", portID)
	}

	allowMsgs := k.GetParams(ctx).AllowMessages
	for _, msg := range msgs {
		if !types.ContainsMsgType(allowMsgs, msg) {
			return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "message type not allowed: %s", sdk.MsgTypeURL(msg))
		}

		// obtain the message signers using the proto signer annotations
		// the msgv2 return value is discarded as it is not used
		signers, _, err := k.cdc.GetMsgSigners(msg)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to obtain message signers for message type %s", sdk.MsgTypeURL(msg))
		}

		for _, signer := range signers {
			// the interchain account address is stored as the string value of the sdk.AccAddress type
			// thus we must cast the signer to a sdk.AccAddress to obtain the comparison value
			// the stored interchain account address must match the signer for every message to be executed
			if interchainAccountAddr != sdk.AccAddress(signer).String() {
				return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "unexpected signer address: expected %s, got %s", interchainAccountAddr, sdk.AccAddress(signer).String())
			}
		}
	}

	return nil
}

// Attempts to get the message handler from the router and if found will then execute the message.
// If the message execution is successful, the proto marshaled message response will be returned.
func (k Keeper) executeMsg(ctx context.Context, msg sdk.Msg) (*codectypes.Any, error) { // TODO: https://github.com/cosmos/ibc-go/issues/7223
	if err := k.MsgRouterService.CanInvoke(ctx, sdk.MsgTypeURL(msg)); err != nil {
		return nil, errorsmod.Wrap(err, icatypes.ErrInvalidRoute.Error())
	}

	res, err := k.MsgRouterService.Invoke(ctx, msg)
	if err != nil {
		return nil, err
	}

	msgResponse, err := codectypes.NewAnyWithValue(res)
	if err != nil {
		return nil, errorsmod.Wrapf(ibcerrors.ErrPackAny, "failed to pack msg response as Any: %T", res)
	}

	return msgResponse, nil
}
