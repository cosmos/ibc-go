package keeper

import (
	"context"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

var _ types.MsgServer = (*Keeper)(nil)

// SendCall defines the handler for the MsgSendCall message.
func (k *Keeper) SendCall(goCtx context.Context, msg *types.MsgSendCall) (*types.MsgSendCallResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	packetData := types.NewGMPPacketData(msg.Sender, msg.Receiver, msg.Salt, msg.Payload, msg.Memo)
	if err := packetData.ValidateBasic(); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to validate %s packet data", types.Version)
	}

	sequence, err := k.sendPacket(ctx, msg.Encoding, msg.SourceClient, msg.TimeoutTimestamp, packetData)
	if err != nil {
		return nil, err
	}

	k.Logger(ctx).Info("IBC send GMP packet", "sender", msg.Sender, "receiver", msg.Receiver)

	return &types.MsgSendCallResponse{Sequence: sequence}, nil
}

func (k *Keeper) sendPacket(ctx sdk.Context, encoding, sourceClient string, timeoutTimestamp uint64, packetData types.GMPPacketData) (uint64, error) {
	if encoding == "" {
		encoding = types.EncodingABI
	}

	data, err := types.MarshalPacketData(&packetData, types.Version, encoding)
	if err != nil {
		return 0, err
	}

	payload := channeltypesv2.NewPayload(types.PortID, types.PortID, types.Version, encoding, data)
	msg := channeltypesv2.NewMsgSendPacket(
		sourceClient, timeoutTimestamp,
		packetData.Sender, payload,
	)

	handler := k.msgRouter.Handler(msg)
	if handler == nil {
		return 0, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "unrecognized packet type: %T", msg)
	}
	res, err := handler(ctx, msg)
	if err != nil {
		return 0, err
	}

	// NOTE: The sdk msg handler creates a new EventManager, so events must be correctly propagated back to the current context
	ctx.EventManager().EmitEvents(res.GetEvents())

	// Each individual sdk.Result has exactly one Msg response. We aggregate here.
	msgResponse := res.MsgResponses[0]
	if msgResponse == nil {
		return 0, errorsmod.Wrapf(ibcerrors.ErrLogic, "got nil Msg response for msg %s", sdk.MsgTypeURL(msg))
	}
	var sendResponse channeltypesv2.MsgSendPacketResponse
	err = proto.Unmarshal(msgResponse.Value, &sendResponse)
	if err != nil {
		return 0, err
	}

	return sendResponse.Sequence, nil
}
