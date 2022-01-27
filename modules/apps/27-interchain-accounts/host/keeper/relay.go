package keeper

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/gogo/protobuf/proto"

	"github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

// OnRecvPacket handles a given interchain accounts packet on a destination host chain
func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet) (txResponse []byte, err error) {
	var data icatypes.InterchainAccountPacketData

	if err := icatypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		// UnmarshalJSON errors are indeterminate and therefore are not wrapped and included in failed acks
		return nil, sdkerrors.Wrapf(icatypes.ErrUnknownDataType, "cannot unmarshal ICS-27 interchain account packet data")
	}

	switch data.Type {
	case icatypes.EXECUTE_TX:
		msgs, err := icatypes.DeserializeCosmosTx(k.cdc, data.Data)
		if err != nil {
			return nil, err
		}

		msgResponses, err := k.executeTx(ctx, packet.SourcePort, packet.DestinationPort, packet.DestinationChannel, msgs)
		if err != nil {
			return nil, err
		}

		txResponse, err := proto.Marshal(&icatypes.TxMsgData{MsgResponses: msgResponses})
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to marshal TxMsgData with Msg responses")
		}

		return txResponse, nil
	default:
		return nil, icatypes.ErrUnknownDataType
	}
}

// authenticateTx ensures the provided msgs contain the correct interchain account signer address retrieved
// from state using the provided controller port identifier
func (k Keeper) authenticateTx(ctx sdk.Context, msgs []sdk.Msg, portID string) error {
	interchainAccountAddr, found := k.GetInterchainAccountAddress(ctx, portID)
	if !found {
		return sdkerrors.Wrapf(icatypes.ErrInterchainAccountNotFound, "failed to retrieve interchain account on port %s", portID)
	}

	allowMsgs := k.GetAllowMessages(ctx)
	for _, msg := range msgs {
		if !types.ContainsMsgType(allowMsgs, msg) {
			return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "message type not allowed: %s", sdk.MsgTypeURL(msg))
		}

		for _, signer := range msg.GetSigners() {
			if interchainAccountAddr != signer.String() {
				return sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "unexpected signer address: expected %s, got %s", interchainAccountAddr, signer.String())
			}
		}
	}

	return nil
}

func (k Keeper) executeTx(ctx sdk.Context, sourcePort, destPort, destChannel string, msgs []sdk.Msg) ([]*codectypes.Any, error) {
	if err := k.authenticateTx(ctx, msgs, sourcePort); err != nil {
		return nil, err
	}

	msgResponses := make([]*codectypes.Any, len(msgs))

	// CacheContext returns a new context with the multi-store branched into a cached storage object
	// writeCache is called only if all msgs succeed, performing state transitions atomically
	cacheCtx, writeCache := ctx.CacheContext()
	for i, msg := range msgs {
		if err := msg.ValidateBasic(); err != nil {
			return nil, err
		}

		msgResponse, err := k.executeMsg(cacheCtx, msg)
		if err != nil {
			return nil, sdkerrors.Wrapf(err, "failed to execute msg at index %d", i)
		}

		msgResponses[i] = msgResponse
	}

	// NOTE: The context returned by CacheContext() creates a new EventManager, so events must be correctly propagated back to the current context
	ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
	writeCache()

	return msgResponses, nil
}

// Attempts to get the message handler from the router and if found will then execute the message
func (k Keeper) executeMsg(ctx sdk.Context, msg sdk.Msg) (*codectypes.Any, error) {
	handler := k.msgRouter.Handler(msg)
	if handler == nil {
		return nil, icatypes.ErrInvalidRoute
	}

	res, err := handler(ctx, msg)
	if err != nil {
		return nil, err
	}

	// NOTE: The sdk msg handler creates a new EventManager, so events must be correctly propagated back to the current context
	ctx.EventManager().EmitEvents(res.GetEvents())

	// NOTE: The format used when constructing the sdk.Result data will change in a future major release of the SDK.
	// This workaround allows for forwards compatibility.
	//
	// The current format uses the directly marshaled Msg response.
	// The future format will used the packed Any of the Msg response.
	//
	// To account for forwards compatibility, we will unmarshal into the Msg response
	// and then pack into an Any
	//
	// The unmarshal and packing code may be removed in future versions when the new SDK approach is avaliable.
	// The res.MsgResponses[0] should be used directly instead of the reconstructured msgResponse

	// Unmarshal the result data into the Msg response
	msgResponse := &sdk.MsgData{}
	if err := proto.Unmarshal(res.Data, msgResponse); err != nil {
		return nil, sdkerrors.Wrap(err, "failed to unmarshal Msg execution result data into sdk.MsgData")
	}

	any, err := codectypes.NewAnyWithValue(msgResponse)
	if err != nil {
		return nil, err
	}

	if any == nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrLogic, "got nil Msg response for msg %s", sdk.MsgTypeURL(msg))
	}

	return any, nil
}
