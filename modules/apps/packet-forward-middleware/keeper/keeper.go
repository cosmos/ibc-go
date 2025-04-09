package keeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
	"github.com/hashicorp/go-metrics"

	corestore "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	coremetrics "github.com/cosmos/ibc-go/v10/modules/core/metrics"
)

var (
	// DefaultTransferPacketTimeoutHeight is the timeout height following IBC defaults
	DefaultTransferPacketTimeoutHeight = clienttypes.Height{
		RevisionNumber: 0,
		RevisionHeight: 0,
	}

	// DefaultForwardTransferPacketTimeoutTimestamp is the timeout timestamp following IBC defaults
	DefaultForwardTransferPacketTimeoutTimestamp = time.Duration(10) * time.Minute
)

// Keeper defines the packet forward middleware keeper
type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.BinaryCodec

	transferKeeper types.TransferKeeper
	channelKeeper  types.ChannelKeeper
	bankKeeper     types.BankKeeper
	ics4Wrapper    porttypes.ICS4Wrapper

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

// NewKeeper creates a new forward Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService corestore.KVStoreService,
	transferKeeper types.TransferKeeper,
	channelKeeper types.ChannelKeeper,
	bankKeeper types.BankKeeper,
	ics4Wrapper porttypes.ICS4Wrapper,
	authority string,
) *Keeper {
	return &Keeper{
		cdc:            cdc,
		storeService:   storeService,
		transferKeeper: transferKeeper,
		channelKeeper:  channelKeeper,
		bankKeeper:     bankKeeper,
		ics4Wrapper:    ics4Wrapper,
		authority:      authority,
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// SetTransferKeeper sets the transferKeeper
func (k *Keeper) SetTransferKeeper(transferKeeper types.TransferKeeper) {
	k.transferKeeper = transferKeeper
}

// Logger returns a module-specific logger.
func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+ibcexported.ModuleName+"-"+types.ModuleName)
}

// moveFundsToUserRecoverableAccount will move the funds from the escrow account to the user recoverable account
// this is only used when the maximum timeouts have been reached or there is an acknowledgement error and the packet is nonrefundable,
// i.e. an operation has occurred to make the original packet funds inaccessible to the user, e.g. a swap.
// We cannot refund the funds back to the original chain, so we move them to an account on this chain that the user can access.
func (k *Keeper) moveFundsToUserRecoverableAccount(
	ctx sdk.Context,
	packet channeltypes.Packet,
	data transfertypes.FungibleTokenPacketData,
	inFlightPacket *types.InFlightPacket,
) error {
	fullDenomPath := data.Denom

	amount, ok := sdkmath.NewIntFromString(data.Amount)
	if !ok {
		return fmt.Errorf("failed to parse amount from packet data for forward recovery: %s", data.Amount)
	}
	denom := transfertypes.ParseDenomTrace(fullDenomPath)
	coin := sdk.NewCoin(denom.IBCDenom(), amount)

	userAccount, err := userRecoverableAccount(inFlightPacket)
	if err != nil {
		return fmt.Errorf("failed to get user recoverable account: %w", err)
	}

	if !transfertypes.SenderChainIsSource(packet.SourcePort, packet.SourceChannel, fullDenomPath) {
		// mint vouchers back to sender
		if err := k.bankKeeper.MintCoins(
			ctx, transfertypes.ModuleName, sdk.NewCoins(coin),
		); err != nil {
			return err
		}

		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, transfertypes.ModuleName, userAccount, sdk.NewCoins(coin)); err != nil {
			panic(fmt.Sprintf("unable to send coins from module to account despite previously minting coins to module account: %v", err))
		}
		return nil
	}

	escrowAddress := transfertypes.GetEscrowAddress(packet.SourcePort, packet.SourceChannel)

	if err := k.bankKeeper.SendCoins(
		ctx, escrowAddress, userAccount, sdk.NewCoins(coin),
	); err != nil {
		return fmt.Errorf("failed to send coins from escrow account to user recoverable account: %w", err)
	}

	// update the total escrow amount for the denom.
	k.unescrowToken(ctx, coin)

	return nil
}

// userRecoverableAccount finds an account on this chain that the original sender of the packet can recover funds from.
// If the destination receiver of the original packet is a valid bech32 address for this chain, we use that address.
// Otherwise, if the sender of the original packet is a valid bech32 address for another chain, we translate that address to this chain.
// Note that for the fallback, the coin type of the source chain sender account must be compatible with this chain.
func userRecoverableAccount(inFlightPacket *types.InFlightPacket) (sdk.AccAddress, error) {
	var originalData transfertypes.FungibleTokenPacketData
	err := transfertypes.ModuleCdc.UnmarshalJSON(inFlightPacket.PacketData, &originalData)
	if err == nil {
		sender, err := sdk.AccAddressFromBech32(originalData.Receiver)
		if err == nil {
			return sender, nil
		}
	}

	_, sender, fallbackErr := bech32.DecodeAndConvert(inFlightPacket.OriginalSenderAddress)
	if fallbackErr == nil {
		return sender, nil
	}

	return nil, fmt.Errorf("failed to decode bech32 addresses: %w", errors.Join(err, fallbackErr))
}

func (k *Keeper) WriteAcknowledgementForForwardedPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	data transfertypes.FungibleTokenPacketData,
	inFlightPacket *types.InFlightPacket,
	ack channeltypes.Acknowledgement,
) error {
	// Lookup module by channel capability
	_, found := k.channelKeeper.GetChannel(ctx, inFlightPacket.RefundPortId, inFlightPacket.RefundChannelId)
	if !found {
		return fmt.Errorf("could not retrieve module from port-id")
	}

	// for forwarded packets, the funds were moved into an escrow account if the denom originated on this chain.
	// On an ack error or timeout on a forwarded packet, the funds in the escrow account
	// should be moved to the other escrow account on the other side or burned.
	if !ack.Success() {
		// If this packet is non-refundable due to some action that took place between the initial ibc transfer and the forward
		// we write a successful ack containing details on what happened regardless of ack error or timeout
		if inFlightPacket.Nonrefundable {
			// we are not allowed to refund back to the source chain.
			// attempt to move funds to user recoverable account on this chain.
			if err := k.moveFundsToUserRecoverableAccount(ctx, packet, data, inFlightPacket); err != nil {
				return err
			}

			ackResult := fmt.Sprintf("packet forward failed after point of no return: %s", ack.GetError())
			newAck := channeltypes.NewResultAcknowledgement([]byte(ackResult))

			return k.ics4Wrapper.WriteAcknowledgement(ctx, channeltypes.Packet{
				Data:               inFlightPacket.PacketData,
				Sequence:           inFlightPacket.RefundSequence,
				SourcePort:         inFlightPacket.PacketSrcPortId,
				SourceChannel:      inFlightPacket.PacketSrcChannelId,
				DestinationPort:    inFlightPacket.RefundPortId,
				DestinationChannel: inFlightPacket.RefundChannelId,
				TimeoutHeight:      clienttypes.MustParseHeight(inFlightPacket.PacketTimeoutHeight),
				TimeoutTimestamp:   inFlightPacket.PacketTimeoutTimestamp,
			}, newAck)
		}

		fullDenomPath := data.Denom
		var err error

		// deconstruct the token denomination into the denomination trace info
		// to determine if the sender is the source chain
		if strings.HasPrefix(data.Denom, "ibc/") {
			fullDenomPath, err = k.transferKeeper.DenomPathFromHash(ctx, data.Denom)
			if err != nil {
				return err
			}
		}

		amount, ok := sdkmath.NewIntFromString(data.Amount)
		if !ok {
			return fmt.Errorf("failed to parse amount from packet data for forward refund: %s", data.Amount)
		}

		denom := transfertypes.ParseDenomTrace(fullDenomPath)
		coin := sdk.NewCoin(denom.IBCDenom(), amount)

		escrowAddress := transfertypes.GetEscrowAddress(packet.SourcePort, packet.SourceChannel)
		refundEscrowAddress := transfertypes.GetEscrowAddress(inFlightPacket.RefundPortId, inFlightPacket.RefundChannelId)

		newToken := sdk.NewCoins(coin)

		// Sender chain is source
		if !denom.HasPrefix(packet.SourcePort, packet.SourceChannel) {
			// funds were moved to escrow account for transfer, so they need to either:
			// - move to the other escrow account, in the case of native denom
			// - burn
			if !denom.HasPrefix(inFlightPacket.RefundPortId, inFlightPacket.RefundChannelId) {
				// transfer funds from escrow account for forwarded packet to escrow account going back for refund.
				if err := k.bankKeeper.SendCoins(
					ctx, escrowAddress, refundEscrowAddress, newToken,
				); err != nil {
					return fmt.Errorf("failed to send coins from escrow account to refund escrow account: %w", err)
				}
			} else {
				// transfer the coins from the escrow account to the module account and burn them.
				if err := k.bankKeeper.SendCoinsFromAccountToModule(
					ctx, escrowAddress, transfertypes.ModuleName, newToken,
				); err != nil {
					return fmt.Errorf("failed to send coins from escrow to module account for burn: %w", err)
				}

				if err := k.bankKeeper.BurnCoins(
					ctx, transfertypes.ModuleName, newToken,
				); err != nil {
					// NOTE: should not happen as the module account was
					// retrieved on the step above and it has enough balance
					// to burn.
					panic(fmt.Sprintf("cannot burn coins after a successful send from escrow account to module account: %v", err))
				}

				k.unescrowToken(ctx, coin)
			}
		} else {
			// Funds in the escrow account were burned,
			// so on a timeout or acknowledgement error we need to mint the funds back to the escrow account.
			if err := k.bankKeeper.MintCoins(ctx, transfertypes.ModuleName, newToken); err != nil {
				return fmt.Errorf("cannot mint coins to the %s module account: %v", transfertypes.ModuleName, err)
			}

			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, transfertypes.ModuleName, refundEscrowAddress, newToken); err != nil {
				return fmt.Errorf("cannot send coins from the %s module to the escrow account %s: %v", transfertypes.ModuleName, refundEscrowAddress, err)
			}

			currentTotalEscrow := k.transferKeeper.GetTotalEscrowForDenom(ctx, coin.GetDenom())
			newTotalEscrow := currentTotalEscrow.Add(coin)
			k.transferKeeper.SetTotalEscrowForDenom(ctx, newTotalEscrow)
		}
	}

	return k.ics4Wrapper.WriteAcknowledgement(ctx, channeltypes.Packet{
		Data:               inFlightPacket.PacketData,
		Sequence:           inFlightPacket.RefundSequence,
		SourcePort:         inFlightPacket.PacketSrcPortId,
		SourceChannel:      inFlightPacket.PacketSrcChannelId,
		DestinationPort:    inFlightPacket.RefundPortId,
		DestinationChannel: inFlightPacket.RefundChannelId,
		TimeoutHeight:      clienttypes.MustParseHeight(inFlightPacket.PacketTimeoutHeight),
		TimeoutTimestamp:   inFlightPacket.PacketTimeoutTimestamp,
	}, ack)
}

// unescrowToken will update the total escrow by deducting the unescrowed token
// from the current total escrow.
func (k *Keeper) unescrowToken(ctx sdk.Context, token sdk.Coin) {
	currentTotalEscrow := k.transferKeeper.GetTotalEscrowForDenom(ctx, token.GetDenom())
	newTotalEscrow := currentTotalEscrow.Sub(token)
	k.transferKeeper.SetTotalEscrowForDenom(ctx, newTotalEscrow)
}

func (k *Keeper) ForwardTransferPacket(
	ctx sdk.Context,
	inFlightPacket *types.InFlightPacket,
	srcPacket channeltypes.Packet,
	srcPacketSender string,
	receiver string,
	metadata *types.ForwardMetadata,
	token sdk.Coin,
	maxRetries uint8,
	timeout time.Duration,
	labels []metrics.Label,
	nonrefundable bool,
) error {
	memo := ""

	// set memo for next transfer with next from this transfer.
	if metadata.Next != nil {
		memoBz, err := json.Marshal(metadata.Next)
		if err != nil {
			k.Logger(ctx).Error("packetForwardMiddleware error marshaling next as JSON",
				"error", err,
			)
			return errorsmod.Wrapf(sdkerrors.ErrJSONMarshal, err.Error())
		}
		memo = string(memoBz)
	}

	msgTransfer := transfertypes.NewMsgTransfer(
		metadata.Port,
		metadata.Channel,
		token,
		receiver,
		metadata.Receiver,
		DefaultTransferPacketTimeoutHeight,
		uint64(ctx.BlockTime().UnixNano())+uint64(timeout.Nanoseconds()),
		memo,
	)

	k.Logger(ctx).Debug("packetForwardMiddleware ForwardTransferPacket",
		"port", metadata.Port, "channel", metadata.Channel,
		"sender", receiver, "receiver", metadata.Receiver,
		"amount", token.Amount.String(), "denom", token.Denom,
	)

	// send tokens to destination
	res, err := k.transferKeeper.Transfer(
		ctx,
		msgTransfer,
	)
	if err != nil {
		k.Logger(ctx).Error("packetForwardMiddleware ForwardTransferPacket error",
			"port", metadata.Port, "channel", metadata.Channel,
			"sender", receiver, "receiver", metadata.Receiver,
			"amount", token.Amount.String(), "denom", token.Denom,
			"error", err,
		)
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	// Store the following information in keeper:
	// key - information about forwarded packet: src_channel (parsedReceiver.Channel), src_port (parsedReceiver.Port), sequence
	// value - information about original packet for refunding if necessary: retries, srcPacketSender, srcPacket.DestinationChannel, srcPacket.DestinationPort
	if inFlightPacket == nil {
		inFlightPacket = &types.InFlightPacket{
			PacketData:            srcPacket.Data,
			OriginalSenderAddress: srcPacketSender,
			RefundChannelId:       srcPacket.DestinationChannel,
			RefundPortId:          srcPacket.DestinationPort,
			RefundSequence:        srcPacket.Sequence,
			PacketSrcPortId:       srcPacket.SourcePort,
			PacketSrcChannelId:    srcPacket.SourceChannel,

			PacketTimeoutTimestamp: srcPacket.TimeoutTimestamp,
			PacketTimeoutHeight:    srcPacket.TimeoutHeight.String(),

			RetriesRemaining: int32(maxRetries),
			Timeout:          uint64(timeout.Nanoseconds()),
			Nonrefundable:    nonrefundable,
		}
	} else {
		inFlightPacket.RetriesRemaining--
	}

	key := types.RefundPacketKey(metadata.Channel, metadata.Port, res.Sequence)
	store := k.storeService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(inFlightPacket)
	store.Set(key, bz)

	defer func() {
		if token.Amount.IsInt64() {
			telemetry.SetGaugeWithLabels(
				[]string{"tx", "msg", "ibc", "transfer"},
				float32(token.Amount.Int64()),
				[]metrics.Label{telemetry.NewLabel(coremetrics.LabelDenom, token.Denom)},
			)
		}

		telemetry.IncrCounterWithLabels(
			[]string{"ibc", types.ModuleName, "send"},
			1,
			labels,
		)
	}()
	return nil
}

// TimeoutShouldRetry returns inFlightPacket and no error if retry should be attempted. Error is returned if IBC refund should occur.
func (k *Keeper) TimeoutShouldRetry(
	ctx sdk.Context,
	packet channeltypes.Packet,
) (*types.InFlightPacket, error) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.RefundPacketKey(packet.SourceChannel, packet.SourcePort, packet.Sequence)

	hasKey, err := store.Has(key)
	if err != nil {
		return nil, err
	}
	if !hasKey {
		// not a forwarded packet, ignore.
		return nil, nil
	}

	bz, err := store.Get(key)
	if err != nil {
		return nil, err
	}
	var inFlightPacket types.InFlightPacket
	k.cdc.MustUnmarshal(bz, &inFlightPacket)

	if inFlightPacket.RetriesRemaining <= 0 {
		k.Logger(ctx).Error("packetForwardMiddleware reached max retries for packet",
			"key", string(key),
			"original-sender-address", inFlightPacket.OriginalSenderAddress,
			"refund-channel-id", inFlightPacket.RefundChannelId,
			"refund-port-id", inFlightPacket.RefundPortId,
		)
		return &inFlightPacket, fmt.Errorf("giving up on packet on channel (%s) port (%s) after max retries",
			inFlightPacket.RefundChannelId, inFlightPacket.RefundPortId)
	}

	return &inFlightPacket, nil
}

func (k *Keeper) RetryTimeout(
	ctx sdk.Context,
	channel, port string,
	data transfertypes.FungibleTokenPacketData,
	inFlightPacket *types.InFlightPacket,
) error {
	// send transfer again
	metadata := &types.ForwardMetadata{
		Receiver: data.Receiver,
		Channel:  channel,
		Port:     port,
	}

	if data.Memo != "" {
		metadata.Next = &types.JSONObject{}
		if err := json.Unmarshal([]byte(data.Memo), metadata.Next); err != nil {
			return fmt.Errorf("error unmarshaling memo json: %w", err)
		}
	}

	amount, ok := sdkmath.NewIntFromString(data.Amount)
	if !ok {
		k.Logger(ctx).Error("packetForwardMiddleware error parsing amount from string for packetforward retry on timeout",
			"original-sender-address", inFlightPacket.OriginalSenderAddress,
			"refund-channel-id", inFlightPacket.RefundChannelId,
			"refund-port-id", inFlightPacket.RefundPortId,
			"retries-remaining", inFlightPacket.RetriesRemaining,
			"amount", data.Amount,
		)
		return fmt.Errorf("error parsing amount from string for packetforward retry: %s", data.Amount)
	}

	ibcDenom := transfertypes.ExtractDenomFromPath(data.Denom).IBCDenom()

	token := sdk.NewCoin(ibcDenom, amount)

	// srcPacket and srcPacketSender are empty because inFlightPacket is non-nil.
	return k.ForwardTransferPacket(
		ctx,
		inFlightPacket,
		channeltypes.Packet{},
		"",
		data.Sender,
		metadata,
		token,
		uint8(inFlightPacket.RetriesRemaining),
		time.Duration(inFlightPacket.Timeout)*time.Nanosecond,
		nil,
		inFlightPacket.Nonrefundable,
	)
}

func (k *Keeper) RemoveInFlightPacket(ctx sdk.Context, packet channeltypes.Packet) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.RefundPacketKey(packet.SourceChannel, packet.SourcePort, packet.Sequence)
	hasKey, err := store.Has(key)
	if err != nil {
		panic(err)
	}
	if !hasKey {
		// not a forwarded packet, ignore.
		return
	}

	// done with packet key now, delete.
	if err := store.Delete(key); err != nil {
		panic(err)
	}
}

// GetAndClearInFlightPacket will fetch an InFlightPacket from the store, remove it if it exists, and return it.
func (k *Keeper) GetAndClearInFlightPacket(
	ctx sdk.Context,
	channel string,
	port string,
	sequence uint64,
) *types.InFlightPacket {
	store := k.storeService.OpenKVStore(ctx)
	key := types.RefundPacketKey(channel, port, sequence)
	hasKey, err := store.Has(key)
	if err != nil {
		panic(err)
	}
	if !hasKey {
		// this is either not a forwarded packet, or it is the final destination for the refund.
		return nil
	}

	bz, err := store.Get(key)
	if err != nil {
		panic(err)
	}

	// done with packet key now, delete.
	store.Delete(key)

	var inFlightPacket types.InFlightPacket
	k.cdc.MustUnmarshal(bz, &inFlightPacket)
	return &inFlightPacket
}

// SendPacket wraps IBC ChannelKeeper's SendPacket function
func (k Keeper) SendPacket(
	ctx sdk.Context,
	sourcePort string, sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (sequence uint64, err error) {
	return k.ics4Wrapper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement wraps IBC ICS4Wrapper WriteAcknowledgement function.
// ICS29 WriteAcknowledgement is used for asynchronous acknowledgements.
func (k *Keeper) WriteAcknowledgement(ctx sdk.Context, packet ibcexported.PacketI, acknowledgement ibcexported.Acknowledgement) error {
	return k.ics4Wrapper.WriteAcknowledgement(ctx, packet, acknowledgement)
}

// WriteAcknowledgement wraps IBC ICS4Wrapper GetAppVersion function.
func (k *Keeper) GetAppVersion(
	ctx sdk.Context,
	portID,
	channelID string,
) (string, bool) {
	return k.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// LookupModuleByChannel wraps ChannelKeeper LookupModuleByChannel function.
func (k *Keeper) GetChannel(ctx sdk.Context, portID, channelID string) (channeltypes.Channel, bool) {
	return k.channelKeeper.GetChannel(ctx, portID, channelID)
}
