package keeper

import (
	"bytes"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

// OnRecvPacket processes a GMP packet.
// Returns the data result of the execution if successful.
func (k *Keeper) OnRecvPacket(
	ctx sdk.Context,
	data *types.GMPPacketData,
	destClient string,
) ([]byte, error) {
	accountID := types.NewAccountIdentifier(destClient, data.Sender, data.Salt)

	ics27Acc, err := k.getOrCreateICS27Account(ctx, &accountID)
	if err != nil {
		return nil, err
	}

	ics27Addr, err := sdk.AccAddressFromBech32(ics27Acc.Address)
	if err != nil {
		return nil, err
	}

	ics27SdkAcc := k.accountKeeper.GetAccount(ctx, ics27Addr)
	if ics27SdkAcc == nil {
		return nil, errorsmod.Wrapf(types.ErrAccountNotFound, "account %s not found", ics27Addr)
	}

	txResponse, err := k.executeTx(ctx, ics27SdkAcc, data.Payload)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to execute ICS27 account transaction")
	}

	return txResponse, nil
}

// executeTx attempts to execute the provided transaction. It begins by authenticating the transaction signer.
// If authentication succeeds, it does basic validation of the messages before attempting to deliver each message
// into state. The state changes will only be committed if all messages in the transaction succeed. Thus the
// execution of the transaction is atomic, all state changes are reverted if a single message fails.
func (k *Keeper) executeTx(ctx sdk.Context, account sdk.AccountI, payload []byte) ([]byte, error) {
	msgs, err := types.DeserializeCosmosTx(k.cdc, payload)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to deserialize ICS27 CosmosTx")
	}

	if err := k.authenticateTx(ctx, account, msgs); err != nil {
		return nil, err
	}

	txMsgData := &sdk.TxMsgData{
		MsgResponses: make([]*codectypes.Any, len(msgs)),
	}

	// CacheContext returns a new context with the multi-store branched into a cached storage object
	// writeCache is called only if all msgs succeed, performing state transitions atomically
	cacheCtx, writeCache := ctx.CacheContext()
	for i, msg := range msgs {
		if m, ok := msg.(sdk.HasValidateBasic); ok {
			if err := m.ValidateBasic(); err != nil {
				return nil, err
			}
		}

		protoAny, err := k.executeMsg(cacheCtx, msg)
		if err != nil {
			ctx.Logger().Error("failed to execute 27-gmp message", "msg", msg, "error", err)
			return nil, err
		}

		txMsgData.MsgResponses[i] = protoAny
	}

	writeCache()

	ctx.Logger().Info("executed 27-gmp transaction", "account", account.GetAddress(), "msgs", msgs)

	txResponse, err := proto.Marshal(txMsgData)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to marshal tx data")
	}

	return txResponse, nil
}

// authenticateTx checks that the transaction is signed by the expected signer.
func (k *Keeper) authenticateTx(_ sdk.Context, account sdk.AccountI, msgs []sdk.Msg) error {
	if len(msgs) == 0 {
		return errorsmod.Wrapf(types.ErrInvalidPayload, "empty message list")
	}

	accountAddr := account.GetAddress()
	for _, msg := range msgs {
		// obtain the message signers using the proto signer annotations
		// the msgv2 return value is discarded as it is not used
		signers, _, err := k.cdc.GetMsgV1Signers(msg)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to obtain message signers for message type %s", sdk.MsgTypeURL(msg))
		}

		for _, signer := range signers {
			// the interchain account address is stored as the string value of the sdk.AccAddress type
			// thus we must cast the signer to a sdk.AccAddress to obtain the comparison value
			// the stored interchain account address must match the signer for every message to be executed
			if !bytes.Equal(signer, accountAddr.Bytes()) {
				return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "unexpected signer address: expected %s, got %s", accountAddr, sdk.AccAddress(signer))
			}
		}
	}

	return nil
}

// Attempts to get the message handler from the router and if found will then execute the message.
// If the message execution is successful, the proto marshaled message response will be returned.
func (k *Keeper) executeMsg(ctx sdk.Context, msg sdk.Msg) (*codectypes.Any, error) {
	handler := k.msgRouter.Handler(msg)
	if handler == nil {
		return nil, types.ErrInvalidMsgRoute
	}

	res, err := handler(ctx, msg)
	if err != nil {
		return nil, err
	}

	// NOTE: The sdk msg handler creates a new EventManager, so events must be correctly propagated back to the current context
	ctx.EventManager().EmitEvents(res.GetEvents())

	// Each individual sdk.Result has exactly one Msg response. We aggregate here.
	msgResponse := res.MsgResponses[0]
	if msgResponse == nil {
		return nil, errorsmod.Wrapf(ibcerrors.ErrLogic, "got nil Msg response for msg %s", sdk.MsgTypeURL(msg))
	}

	return msgResponse, nil
}
