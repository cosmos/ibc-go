package fee

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var (
	_ porttypes.IBCModuleV2 = (*IBCModuleV2)(nil)
)

// IBCModuleV2 implements the ICS26 interface for transfer given the transfer keeper.
type IBCModuleV2 struct {
	keeper keeper.Keeper
}

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModuleV2(k keeper.Keeper) IBCModuleV2 {
	return IBCModuleV2{
		keeper: k,
	}
}

// OnSendPacketV2 implements the IBCModuleV2 interface.
func (im IBCModuleV2) OnSendPacketV2(
	ctx sdk.Context,
	portID string,
	channelID string,
	sequence uint64,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	payload channeltypes.Payload,
	signer sdk.AccAddress,
) error {
	if im.keeper.IsLocked(ctx) {
		return types.ErrFeeModuleLocked
	}

	// TODO: validate encoding, validate version
	if payload.Version != types.Version {
		return errorsmod.Wrapf(types.ErrInvalidVersion, "expected %s, got %s", types.Version, payload.Version)
	}

	// TODO: unmarshal payload with encoding, now: assume protobuf
	var data types.PacketData
	if err := proto.Unmarshal(payload.Value, &data); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot %s unmarshal %s packet data: %s", payload.Encoding, payload.Version, err.Error())
	}

	if err := im.keeper.IsSendEnabledCoins(ctx, data.PacketFee.Fee.Total()...); err != nil {
		return err
	}

	// NOTE: this check is now slightly redundant considering the address has gone through sigverify ante and core (we should still do the check defensively)
	if im.keeper.BlockedAddr(signer) {
		return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "%s is not allowed to escrow fees", signer)
	}

	packetID := channeltypes.NewPacketID(portID, channelID, sequence)
	// TODO: packet data structure and checks on packetFee, e.g. signer == refundAddress?
	// packetFee := types.NewPacketFee(data.PacketFee.Fee, msg.Signer, msg.Relayers)

	return im.keeper.EscrowPacketFeeV2(ctx, packetID, data.PacketFee)
}

// OnRecvPacketV2 implements the IBCModule interface. A successful acknowledgement
// is returned if the packet data is successfully decoded and the receive application
// logic returns without error.
// A nil acknowledgement may be returned when using the packet forwarding feature. This signals to core IBC that the acknowledgement will be written asynchronously.
func (im IBCModuleV2) OnRecvPacketV2(
	ctx sdk.Context,
	packet channeltypes.PacketV2,
	payload channeltypes.Payload,
	relayer sdk.AccAddress,
) channeltypes.RecvPacketResult {
	// TODO: should we do something with found value here (dislike the silent nature of an empty string encoded into ack)
	// e.g.
	// Use an explicit "empty fee acknowledgement" when no counterparty address is registered for forward relay payment (delivery of MsgRecvPacket).
	// forwardRelayer, found := im.keeper.GetCounterpartyPayeeAddress(ctx, relayer.String(), packet.DestinationChannel)
	// if !found {
	//     feeAcknowledgement = types.NewEmptyFeeAcknowledgement()
	// }

	// forwardRelayer, _ := im.keeper.GetCounterpartyPayeeAddress(ctx, relayer.String(), packet.DestinationChannel)
	// feeAcknowledgement := types.NewFeeAcknowledgement(forwardRelayer)

	// NOTE: this is functionality equivalent to above where found bool is discarded
	var feeAcknowledgement types.FeeAcknowledgement
	if forwardRelayer, found := im.keeper.GetCounterpartyPayeeAddress(ctx, relayer.String(), packet.DestinationChannel); found {
		feeAcknowledgement = types.NewFeeAcknowledgement(forwardRelayer)
	}

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return channeltypes.RecvPacketResult{Status: channeltypes.PacketStatus_Success, Acknowledgement: feeAcknowledgement.Acknowledgement()}
}

func (im IBCModuleV2) OnAcknowledgementPacketV2(
	ctx sdk.Context,
	packet channeltypes.PacketV2,
	payload channeltypes.Payload,
	result channeltypes.RecvPacketResult, // TODO: why not just acknowledgement data, status isn't needed here... depends on whats outputted in recv packet events
	relayer sdk.AccAddress,
) error {
	var ack types.FeeAcknowledgement
	if err := json.Unmarshal(result.Acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot unmarshal ICS-29 incentivized packet acknowledgement %v: %s", ack, err)
	}

	if im.keeper.IsLocked(ctx) {
		// if the fee keeper is locked then fee logic should be skipped
		// this may occur in the presence of a severe bug which leads to invalid state
		// the fee keeper will be unlocked after manual intervention
		// the acknowledgement has been unmarshalled into an ics29 acknowledgement
		// since the counterparty is still sending incentivized acknowledgements
		// for fee enabled channels
		//
		// Please see ADR 004 for more information.
		return nil
	}

	packetID := channeltypes.NewPacketID(packet.SourcePort, packet.SourceChannel, packet.Sequence)
	feesInEscrow, found := im.keeper.GetFeesInEscrow(ctx, packetID)
	if !found {
		return nil
	}

	payee, found := im.keeper.GetPayeeAddress(ctx, relayer.String(), packet.SourceChannel)
	if !found {
		payee = relayer.String()
	}

	payeeAddr, err := sdk.AccAddressFromBech32(payee)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to create sdk.Address from payee: %s", payee)
	}

	im.keeper.DistributePacketFeesOnAcknowledgement(ctx, ack.ForwardRelayerAddress, payeeAddr, feesInEscrow.PacketFees, packetID)
	return nil
}

func (im IBCModuleV2) OnTimeoutPacketV2(
	ctx sdk.Context,
	packet channeltypes.PacketV2,
	payload channeltypes.Payload,
	relayer sdk.AccAddress,
) error {
	// if the fee keeper is locked then fee logic should be skipped
	// this may occur in the presence of a severe bug which leads to invalid state
	// the fee keeper will be unlocked after manual intervention
	//
	// Please see ADR 004 for more information.
	if im.keeper.IsLocked(ctx) {
		return nil
	}

	packetID := channeltypes.NewPacketID(packet.SourcePort, packet.SourceChannel, packet.Sequence)
	feesInEscrow, found := im.keeper.GetFeesInEscrow(ctx, packetID)
	if !found {
		return nil // no escrowed fees for packet ID
	}

	payee, found := im.keeper.GetPayeeAddress(ctx, relayer.String(), packet.SourceChannel)
	if !found {
		payee = relayer.String()
	}

	payeeAddr, err := sdk.AccAddressFromBech32(payee)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to create sdk.Address from payee: %s", payee)
	}

	im.keeper.DistributePacketFeesOnTimeout(ctx, payeeAddr, feesInEscrow.PacketFees, packetID)
	return nil
}
