package packetforward

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-metrics"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	_ porttypes.Middleware              = &IBCMiddleware{}
	_ porttypes.PacketUnmarshalerModule = &IBCMiddleware{}
)

// IBCMiddleware implements the ICS26 callbacks for the forward middleware given the
// forward keeper and the underlying application.
type IBCMiddleware struct {
	app    porttypes.PacketUnmarshalerModule
	keeper *keeper.Keeper

	retriesOnTimeout uint8
	forwardTimeout   time.Duration
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application.
func NewIBCMiddleware(k *keeper.Keeper, retriesOnTimeout uint8, forwardTimeout time.Duration) *IBCMiddleware {
	return &IBCMiddleware{
		keeper:           k,
		retriesOnTimeout: retriesOnTimeout,
		forwardTimeout:   forwardTimeout,
	}
}

// OnChanOpenInit implements the IBCModule interface.
func (im *IBCMiddleware) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, counterparty channeltypes.Counterparty, version string) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
}

// OnChanOpenTry implements the IBCModule interface.
func (im *IBCMiddleware) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterparty channeltypes.Counterparty, counterpartyVersion string) (string, error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCModule interface.
func (im *IBCMiddleware) OnChanOpenAck(ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCModule interface.
func (im *IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCModule interface.
func (im *IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCModule interface.
func (im *IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// UnmarshalPacketData implements PacketDataUnmarshaler.
func (im *IBCMiddleware) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (any, string, error) {
	return im.app.UnmarshalPacketData(ctx, portID, channelID, bz)
}

func getDenomForThisChain(port, channel, counterpartyPort, counterpartyChannel string, denom transfertypes.Denom) string {
	if denom.HasPrefix(counterpartyPort, counterpartyChannel) {
		// unwind denom
		denom.Trace = denom.Trace[1:]
		if len(denom.Trace) == 0 {
			// denom is now unwound back to native denom
			return denom.Path()
		}
		// denom is still IBC denom
		return denom.IBCDenom()
	}
	// append port and channel from this chain to denom
	trace := []transfertypes.Hop{transfertypes.NewHop(port, channel)}
	denom.Trace = append(trace, denom.Trace...)

	return denom.IBCDenom()
}

// getBoolFromAny returns the bool value is any is a valid bool, otherwise false.
func getBoolFromAny(value any) bool {
	if value == nil {
		return false
	}
	boolVal, ok := value.(bool)
	if !ok {
		return false
	}
	return boolVal
}

// GetReceiver returns the receiver address for a given channel and original sender.
// it overrides the receiver address to be a hash of the channel/origSender so that
// the receiver address is deterministic and can be used to identify the sender on the
// initial chain.
func GetReceiver(channel string, originalSender string) (string, error) {
	senderStr := fmt.Sprintf("%s/%s", channel, originalSender)
	senderHash32 := address.Hash(types.ModuleName, []byte(senderStr))
	sender := sdk.AccAddress(senderHash32[:20])
	bech32Prefix := sdk.GetConfig().GetBech32AccountAddrPrefix()
	return sdk.Bech32ifyAddressBytes(bech32Prefix, sender)
}

// newErrorAcknowledgement returns an error that identifies PFM and provides the error.
// It's okay if these errors are non-deterministic, because they will not be committed to state, only emitted as events.
func newErrorAcknowledgement(err error) channeltypes.Acknowledgement {
	return channeltypes.Acknowledgement{
		Response: &channeltypes.Acknowledgement_Error{
			Error: fmt.Sprintf("packet-forward-middleware error: %s", err.Error()),
		},
	}
}

// OnRecvPacket checks the memo field on this packet and if the metadata inside's root key indicates this packet
// should be handled by the swap middleware it attempts to perform a swap. If the swap is successful
// the underlying application's OnRecvPacket callback is invoked, an ack error is returned otherwise.
func (im *IBCMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	logger := im.keeper.Logger(ctx)

	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		logger.Debug(fmt.Sprintf("packetForwardMiddleware OnRecvPacket payload is not a FungibleTokenPacketData: %s", err.Error()))
		return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	transferDetail, err := transfertypes.PacketDataV1ToV2(data)
	if err != nil {
		logger.Error(fmt.Sprintf("packetForwardMiddleware OnRecvPacket could not convert FungibleTokenPacketData to InternalRepresentation: %s", err.Error()))
		return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}

	logger.Debug("packetForwardMiddleware OnRecvPacket",
		"sequence", packet.Sequence,
		"src-channel", packet.SourceChannel,
		"src-port", packet.SourcePort,
		"dst-channel", packet.DestinationChannel,
		"dst-port", packet.DestinationPort,
		"amount", data.Amount,
		"denom", data.Denom,
		"memo", data.Memo,
	)

	packetMetadata, isPFM, err := types.GetPacketMetadataFromPacketdata(data)
	if err != nil && !isPFM {
		// not a packet that should be forwarded
		logger.Debug("packetForwardMiddleware OnRecvPacket forward metadata does not exist")
		return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
	}
	if err != nil && isPFM {
		logger.Error("packetForwardMiddleware OnRecvPacket error parsing forward metadata", "error", err)
		return newErrorAcknowledgement(fmt.Errorf("error parsing forward metadata: %w", err))
	}

	metadata := packetMetadata.Forward

	goCtx := ctx.Context()
	nonrefundable := getBoolFromAny(goCtx.Value(types.NonrefundableKey{}))

	if err := metadata.Validate(); err != nil {
		logger.Error("packetForwardMiddleware OnRecvPacket forward metadata is invalid", "error", err)
		return newErrorAcknowledgement(err)
	}

	// override the receiver so that senders cannot move funds through arbitrary addresses.
	overrideReceiver, err := GetReceiver(packet.DestinationChannel, data.Sender)
	if err != nil {
		logger.Error("packetForwardMiddleware OnRecvPacket failed to construct override receiver", "error", err)
		return newErrorAcknowledgement(fmt.Errorf("failed to construct override receiver: %w", err))
	}

	if err := im.receiveFunds(ctx, channelVersion, packet, data, overrideReceiver, relayer); err != nil {
		logger.Error("packetForwardMiddleware OnRecvPacket error receiving packet", "error", err)
		return newErrorAcknowledgement(fmt.Errorf("error receiving packet: %w", err))
	}

	// if this packet's token denom is already the base denom for some native token on this chain,
	// we do not need to do any further composition of the denom before forwarding the packet
	denomOnThisChain := getDenomForThisChain(packet.DestinationPort, packet.DestinationChannel, packet.SourcePort, packet.SourceChannel, transferDetail.Token.Denom)

	amountInt, ok := sdkmath.NewIntFromString(transferDetail.Token.Amount)
	if !ok {
		logger.Error("packetForwardMiddleware OnRecvPacket error parsing amount for forward", "amount", transferDetail.Token.Amount)
		return newErrorAcknowledgement(fmt.Errorf("error parsing amount for forward: %s", transferDetail.Token.Amount))
	}

	token := sdk.NewCoin(denomOnThisChain, amountInt)

	timeout := metadata.Timeout

	if timeout.Nanoseconds() <= 0 {
		timeout = im.forwardTimeout
	}

	var retries uint8
	if metadata.Retries != nil {
		retries = *metadata.Retries
	} else {
		retries = im.retriesOnTimeout
	}

	err = im.keeper.ForwardTransferPacket(ctx, nil, packet, data.Sender, overrideReceiver, metadata, token, retries, timeout, []metrics.Label{}, nonrefundable)
	if err != nil {
		logger.Error("packetForwardMiddleware OnRecvPacket error forwarding packet", "error", err)
		return newErrorAcknowledgement(err)
	}

	// returning nil ack will prevent WriteAcknowledgement from occurring for forwarded packet.
	// This is intentional so that the acknowledgement will be written later based on the ack/timeout of the forwarded packet.
	return nil
}

// receiveFunds receives funds from the packet into the override receiver
// address and returns an error if the funds cannot be received.
func (im *IBCMiddleware) receiveFunds(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, data transfertypes.FungibleTokenPacketData, overrideReceiver string, relayer sdk.AccAddress) error {
	overrideData := data
	overrideData.Receiver = overrideReceiver
	overrideData.Memo = "" // Memo explicitly emptied.

	overrideDataBz := transfertypes.ModuleCdc.MustMarshalJSON(&overrideData)

	overridePacket := packet
	overridePacket.Data = overrideDataBz // Override data.
	ack := im.app.OnRecvPacket(ctx, channelVersion, overridePacket, relayer)
	if ack == nil {
		return errors.New("ack is nil")
	}

	if !ack.Success() {
		return fmt.Errorf("ack error: %s", string(ack.Acknowledgement()))
	}

	return nil
}

// OnAcknowledgementPacket implements the IBCModule interface.
func (im *IBCMiddleware) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		im.keeper.Logger(ctx).Error("packetForwardMiddleware error parsing packet data from ack packet",
			"sequence", packet.Sequence,
			"src-channel", packet.SourceChannel,
			"src-port", packet.SourcePort,
			"dst-channel", packet.DestinationChannel,
			"dst-port", packet.DestinationPort,
			"error", err,
		)
		return im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
	}
	transferDetail, err := transfertypes.PacketDataV1ToV2(data)
	if err != nil {
		im.keeper.Logger(ctx).Error("packetForwardMiddleware error converting FungibleTokenPacket to InternalRepresentation")
		return im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
	}

	im.keeper.Logger(ctx).Debug("packetForwardMiddleware OnAcknowledgementPacket",
		"sequence", packet.Sequence,
		"src-channel", packet.SourceChannel,
		"src-port", packet.SourcePort,
		"dst-channel", packet.DestinationChannel,
		"dst-port", packet.DestinationPort,
		"amount", data.Amount,
		"denom", data.Denom,
	)

	var ack channeltypes.Acknowledgement
	if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet acknowledgement: %v", err)
	}

	inFlightPacket, err := im.keeper.GetInflightPacket(ctx, packet)
	if err != nil {
		return err
	}

	if inFlightPacket != nil {
		im.keeper.RemoveInFlightPacket(ctx, packet)
		// this is a forwarded packet, so override handling to avoid refund from being processed.
		return im.keeper.WriteAcknowledgementForForwardedPacket(ctx, packet, transferDetail, inFlightPacket, ack)
	}

	return im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCModule interface.
func (im *IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		im.keeper.Logger(ctx).Error("packetForwardMiddleware error parsing packet data from timeout packet",
			"sequence", packet.Sequence,
			"src-channel", packet.SourceChannel,
			"src-port", packet.SourcePort,
			"dst-channel", packet.DestinationChannel,
			"dst-port", packet.DestinationPort,
			"error", err,
		)
		return im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
	}

	transferDetail, err := transfertypes.PacketDataV1ToV2(data)
	if err != nil {
		im.keeper.Logger(ctx).Error("packetForwardMiddleware error converting FungibleTokenPacket to InternalRepresentation")
		return im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
	}

	im.keeper.Logger(ctx).Debug("packetForwardMiddleware OnTimeoutPacket",
		"sequence", packet.Sequence,
		"src-channel", packet.SourceChannel,
		"src-port", packet.SourcePort,
		"dst-channel", packet.DestinationChannel,
		"dst-port", packet.DestinationPort,
		"amount", data.Amount,
		"denom", data.Denom,
	)

	inFlightPacket, err := im.keeper.TimeoutShouldRetry(ctx, packet)
	if inFlightPacket != nil {
		im.keeper.RemoveInFlightPacket(ctx, packet)
		if err != nil {
			// this is a forwarded packet, so override handling to avoid refund from being processed on this chain.
			// WriteAcknowledgement with proxied ack to return success/fail to previous chain.
			return im.keeper.WriteAcknowledgementForForwardedPacket(ctx, packet, transferDetail, inFlightPacket, newErrorAcknowledgement(err))
		}
		// timeout should be retried. In order to do that, we need to handle this timeout to refund on this chain first.
		if err := im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer); err != nil {
			return err
		}
		return im.keeper.RetryTimeout(ctx, packet.SourceChannel, packet.SourcePort, transferDetail, inFlightPacket)
	}

	return im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

// SendPacket implements the ICS4 Wrapper interface.
func (im *IBCMiddleware) SendPacket(ctx sdk.Context, sourcePort, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error) {
	return im.keeper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement implements the ICS4 Wrapper interface.
func (im *IBCMiddleware) WriteAcknowledgement(ctx sdk.Context, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return im.keeper.WriteAcknowledgement(ctx, packet, ack)
}

func (im *IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.keeper.GetAppVersion(ctx, portID, channelID)
}

func (im *IBCMiddleware) SetICS4Wrapper(wrapper porttypes.ICS4Wrapper) {
	if wrapper == nil {
		panic("ICS4Wrapper cannot be nil")
	}
	im.keeper.WithICS4Wrapper(wrapper)
}

func (im *IBCMiddleware) SetUnderlyingApplication(app porttypes.IBCModule) {
	if im.app != nil {
		panic("underlying application already set")
	}
	// the underlying application must implement the PacketUnmarshalerModule interface
	pdApp, ok := app.(porttypes.PacketUnmarshalerModule)
	if !ok {
		panic(fmt.Errorf("underlying application must implement PacketUnmarshalerModule, got %T", app))
	}
	im.app = pdApp
}
