package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/events"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/telemetry"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	clienttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/02-client/v2/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
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

	packetData := types.NewFungibleTokenPacketData(token.Denom.Path(), token.Amount, sender.String(), msg.Receiver, msg.Memo)

	if err := packetData.ValidateBasic(); err != nil {
		return nil, errorsmod.Wrapf(err, "failed to validate %s packet data", types.V1)
	}

	// if a eureka counterparty exists with source channel, then use IBC V2 protocol
	// otherwise use IBC V1 protocol
	var (
		channel            channeltypes.Channel
		counterparty       clienttypesv2.CounterpartyInfo
		destinationPort    string
		destinationChannel string
		isIBCV2            bool
	)
	counterparty, isIBCV2 = k.clientKeeperV2.GetClientCounterparty(ctx, msg.SourceChannel)
	if !isIBCV2 {
		var found bool
		channel, found = k.channelKeeper.GetChannel(ctx, msg.SourcePort, msg.SourceChannel)
		if !found {
			return nil, errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", msg.SourcePort, msg.SourceChannel)
		}
	}

	var sequence uint64
	if isIBCV2 {
		sequence, err = k.transferV2Packet(ctx, msg.Encoding, msg.SourceChannel, msg.TimeoutTimestamp, packetData)
		destinationPort = types.PortID
		destinationChannel = counterparty.ClientId
	} else {
		sequence, err = k.transferV1Packet(ctx, msg.SourceChannel, token, msg.TimeoutHeight, msg.TimeoutTimestamp, packetData)
		destinationPort = channel.Counterparty.PortId
		destinationChannel = channel.Counterparty.ChannelId
	}
	if err != nil {
		return nil, err
	}

	telemetry.ReportTransfer(types.PortID, msg.SourceChannel, destinationPort, destinationChannel, token)

	k.Logger(ctx).Info("IBC fungible token transfer", "token", coin, "sender", msg.Sender, "receiver", msg.Receiver)

	return &types.MsgTransferResponse{Sequence: sequence}, nil
}

func (k Keeper) transferV1Packet(ctx sdk.Context, sourceChannel string, token types.Token, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, packetData types.FungibleTokenPacketData) (uint64, error) {
	if err := k.SendTransfer(ctx, types.PortID, sourceChannel, token, sdk.AccAddress(packetData.Sender)); err != nil {
		return 0, err
	}

	packetDataBytes := packetData.GetBytes()
	sequence, err := k.ics4Wrapper.SendPacket(ctx, types.PortID, sourceChannel, timeoutHeight, timeoutTimestamp, packetDataBytes)
	if err != nil {
		return 0, err
	}

	events.EmitTransferEvent(ctx, packetData.Sender, packetData.Receiver, token, packetData.Memo)

	return sequence, nil
}

func (k Keeper) transferV2Packet(ctx sdk.Context, encoding, sourceChannel string, timeoutTimestamp uint64, packetData types.FungibleTokenPacketData) (uint64, error) {
	if encoding == "" {
		encoding = types.EncodingJSON
	}

	data, err := types.MarshalPacketData(packetData, types.V1, encoding)
	if err != nil {
		return 0, err
	}

	payload := channeltypesv2.NewPayload(
		types.PortID, types.PortID,
		types.V1, encoding, data,
	)
	msg := channeltypesv2.NewMsgSendPacket(
		sourceChannel, timeoutTimestamp,
		packetData.Sender, payload,
	)

	res, err := k.channelKeeperV2.SendPacket(ctx, msg)
	if err != nil {
		return 0, err
	}
	return res.Sequence, nil
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
