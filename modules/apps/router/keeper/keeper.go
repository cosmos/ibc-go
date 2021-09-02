package keeper

import (
	"time"

	"github.com/armon/go-metrics"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/ibc-go/modules/apps/router/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	coretypes "github.com/cosmos/ibc-go/modules/core/types"
)

// Keeper defines the IBC fungible transfer keeper
type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace

	transferKeeper types.TransferKeeper
	bankKeeper     types.BankKeeper
}

// NewKeeper creates a new 29-fee Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace,
	transferKeeper types.TransferKeeper, bankKeeper types.BankKeeper,
) Keeper {

	return Keeper{
		cdc:            cdc,
		storeKey:       key,
		transferKeeper: transferKeeper,
		paramSpace:     paramSpace,
		bankKeeper:     bankKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+host.ModuleName+"-"+types.ModuleName)
}

func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, data transfertypes.FungibleTokenPacketData, receiver sdk.AccAddress, finalDest, port, channel string) error {
	// validate packet data upon receiving
	if err := data.ValidateBasic(); err != nil {
		return err
	}

	if !k.transferKeeper.GetReceiveEnabled(ctx) {
		return transfertypes.ErrReceiveDisabled
	}

	if !k.transferKeeper.GetSendEnabled(ctx) {
		return transfertypes.ErrSendDisabled
	}

	labels := []metrics.Label{
		telemetry.NewLabel(coretypes.LabelSourcePort, packet.GetSourcePort()),
		telemetry.NewLabel(coretypes.LabelSourceChannel, packet.GetSourceChannel()),
	}

	if transfertypes.ReceiverChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), data.Denom) {

		// remove prefix added by sender chain
		voucherPrefix := transfertypes.GetDenomPrefix(packet.GetSourcePort(), packet.GetSourceChannel())
		unprefixedDenom := data.Denom[len(voucherPrefix):]

		// coin denomination used in sending from the escrow address
		denom := unprefixedDenom

		// The denomination used to send the coins is either the native denom or the hash of the path
		// if the denomination is not native.
		denomTrace := transfertypes.ParseDenomTrace(unprefixedDenom)
		if denomTrace.Path != "" {
			denom = denomTrace.IBCDenom()
		}
		token := sdk.NewCoin(denom, sdk.NewIntFromUint64(data.Amount))

		// unescrow tokens
		if err := k.bankKeeper.SendCoins(ctx, transfertypes.GetEscrowAddress(packet.GetDestPort(), packet.GetDestChannel()), receiver, sdk.NewCoins(token)); err != nil {
			// NOTE: this error is only expected to occur given an unexpected bug or a malicious
			// counterparty module. The bug may occur in bank or any part of the code that allows
			// the escrow address to be drained. A malicious counterparty module could drain the
			// escrow address by allowing more tokens to be sent back then were escrowed.
			return sdkerrors.Wrap(err, "unable to unescrow tokens, this may be caused by a malicious counterparty module or a bug: please open an issue on counterparty module")
		}

		defer func() {
			telemetry.SetGaugeWithLabels(
				[]string{"ibc", types.ModuleName, "packet", "receive"},
				float32(data.Amount),
				[]metrics.Label{telemetry.NewLabel(coretypes.LabelDenom, unprefixedDenom)},
			)

			telemetry.IncrCounterWithLabels(
				[]string{"ibc", types.ModuleName, "receive"},
				1,
				append(
					labels, telemetry.NewLabel(coretypes.LabelSource, "true"),
				),
			)
		}()

		if err := k.ForwardTransferPacket(ctx, receiver, token, port, channel, finalDest, labels); err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
		}

		return nil
	}

	// sender chain is the source, mint vouchers

	// In middleware case,
	// split out reciever address into parts
	// allow processing from standard ics20 module
	// // app.OnRecievePacket()
	// // our custom logic
	// after this happens then we create our packet and pay fees

	// since SendPacket did not prefix the denomination, we must prefix denomination here
	sourcePrefix := transfertypes.GetDenomPrefix(packet.GetDestPort(), packet.GetDestChannel())
	// NOTE: sourcePrefix contains the trailing "/"
	prefixedDenom := sourcePrefix + data.Denom

	// construct the denomination trace from the full raw denomination
	denomTrace := transfertypes.ParseDenomTrace(prefixedDenom)

	traceHash := denomTrace.Hash()
	if !k.transferkeeper.HasDenomTrace(ctx, traceHash) {
		k.transferkeeper.SetDenomTrace(ctx, denomTrace)
	}

	voucherDenom := denomTrace.IBCDenom()
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDenomTrace,
			sdk.NewAttribute(types.AttributeKeyTraceHash, traceHash.String()),
			sdk.NewAttribute(types.AttributeKeyDenom, voucherDenom),
		),
	)

	voucher := sdk.NewCoin(voucherDenom, sdk.NewIntFromUint64(data.Amount))

	// mint new tokens if the source of the transfer is the same chain
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(voucher)); err != nil {
		return err
	}

	// send to receiver
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, sdk.NewCoins(voucher)); err != nil {
		return err
	}

	defer func() {
		telemetry.SetGaugeWithLabels(
			[]string{"ibc", types.ModuleName, "packet", "receive"},
			float32(data.Amount),
			[]metrics.Label{telemetry.NewLabel(coretypes.LabelDenom, data.Denom)},
		)

		telemetry.IncrCounterWithLabels(
			[]string{"ibc", types.ModuleName, "receive"},
			1,
			append(
				labels, telemetry.NewLabel(coretypes.LabelSource, "false"),
			),
		)
	}()

	if finalDest != "" && port != "" && channel != "" {
		if err := k.ForwardTransferPacket(ctx, receiver, voucher, port, channel, finalDest, labels); err != nil {
			return err
		}
	}

	return nil

}

func (k Keeper) ForwardTransferPacket(ctx sdk.Context, receiver sdk.AccAddress, token sdk.Coin, port, channel, finalDest string, labels []metrics.Label) error {
	feeAmount := token.Amount.ToDec().Mul(k.GetFeePercentage(ctx)).RoundInt()
	packetAmount := token.Amount.Sub(feeAmount)
	feeCoins := sdk.Coins{sdk.NewCoin(token.Denom, feeAmount)}
	packetCoin := sdk.NewCoin(token.Denom, packetAmount)

	// pay fees
	if feeAmount.IsPositive() {
		// TODO: maybe community pool
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, receiver, authtypes.FeeCollectorName, feeCoins); err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
		}
	}

	// send tokens to destination
	if err := k.transferKeeper.SendTransfer(ctx, port, channel, packetCoin, receiver, finalDest, clienttypes.Height{0, 0}, uint64(ctx.BlockTime().Add(30*time.Minute).UnixNano())); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	defer func() {
		telemetry.SetGaugeWithLabels(
			[]string{"tx", "msg", "ibc", "transfer"},
			float32(token.Amount.Int64()),
			[]metrics.Label{telemetry.NewLabel(coretypes.LabelDenom, token.Denom)},
		)

		telemetry.IncrCounterWithLabels(
			[]string{"ibc", types.ModuleName, "send"},
			1,
			labels,
		)
	}()
	return nil
}
