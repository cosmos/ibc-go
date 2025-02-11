package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/events"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/telemetry"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var _ types.MsgServer = (*Keeper)(nil)

// Transfer defines an rpc handler method for MsgTransfer.
func (k Keeper) Transfer(goCtx context.Context, msg *types.MsgTransfer) (*types.MsgTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.GetParams(ctx).SendEnabled {
		return nil, types.ErrSendDisabled
	}

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	channel, found := k.channelKeeper.GetChannel(ctx, msg.SourcePort, msg.SourceChannel)
	if !found {
		return nil, errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", msg.SourcePort, msg.SourceChannel)
	}

	appVersion, found := k.ics4Wrapper.GetAppVersion(ctx, msg.SourcePort, msg.SourceChannel)
	if !found {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "application version not found for source port: %s and source channel: %s", msg.SourcePort, msg.SourceChannel)
	}

	coin := msg.Token

	// Using types.UnboundedSpendLimit allows us to send the entire balance of a given denom.
	if coin.Amount.Equal(types.UnboundedSpendLimit()) {
		coin.Amount = k.BankKeeper.SpendableCoin(ctx, sender, coin.Denom).Amount
		if coin.Amount.IsZero() {
			return nil, errorsmod.Wrapf(types.ErrInvalidAmount, "empty spendable balance for %s", coin.Denom)
		}
	}

	token, err := k.TokenFromCoin(ctx, coin)
	if err != nil {
		return nil, err
	}

	if err := k.SendTransfer(ctx, msg.SourcePort, msg.SourceChannel, token, sender); err != nil {
		return nil, err
	}

	packetDataBytes, err := createPacketDataBytesFromVersion(
		appVersion, sender.String(), msg.Receiver, msg.Memo, token,
	)
	if err != nil {
		return nil, err
	}

	sequence, err := k.ics4Wrapper.SendPacket(ctx, msg.SourcePort, msg.SourceChannel, msg.TimeoutHeight, msg.TimeoutTimestamp, packetDataBytes)
	if err != nil {
		return nil, err
	}

	events.EmitTransferEvent(ctx, sender.String(), msg.Receiver, token, msg.Memo)

	destinationPort := channel.Counterparty.PortId
	destinationChannel := channel.Counterparty.ChannelId
	telemetry.ReportTransfer(msg.SourcePort, msg.SourceChannel, destinationPort, destinationChannel, token)

	k.Logger(ctx).Info("IBC fungible token transfer", "token", coin, "sender", msg.Sender, "receiver", msg.Receiver)

	return &types.MsgTransferResponse{Sequence: sequence}, nil
}

// UpdateParams defines an rpc handler method for MsgUpdateParams. Updates the ibc-transfer module's parameters.
func (k Keeper) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Signer {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Signer)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}
