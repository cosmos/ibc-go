package gmp

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/internal/events"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/api"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

var (
	_ api.IBCModule             = (*IBCModule)(nil)
	_ api.PacketDataUnmarshaler = (*IBCModule)(nil)
)

// IBCModule implements the ICS26 interface for transfer given the transfer keeper.
type IBCModule struct {
	keeper *keeper.Keeper
}

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(k *keeper.Keeper) *IBCModule {
	return &IBCModule{
		keeper: k,
	}
}

func (*IBCModule) OnSendPacket(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress) error {
	if payload.SourcePort != types.PortID || payload.DestinationPort != types.PortID {
		return errorsmod.Wrapf(channeltypesv2.ErrInvalidPacket, "payload port ID is invalid: expected %s, got sourcePort: %s destPort: %s", types.PortID, payload.SourcePort, payload.DestinationPort)
	}
	if !clienttypes.IsValidClientID(sourceChannel) || !clienttypes.IsValidClientID(destinationChannel) {
		return errorsmod.Wrapf(channeltypesv2.ErrInvalidPacket, "client IDs must be in valid format: {string}-{number}")
	}

	data, err := types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return err
	}

	if err := data.ValidateBasic(); err != nil {
		return errorsmod.Wrapf(err, "failed to validate %s packet data", types.Version)
	}

	sender, err := sdk.AccAddressFromBech32(data.Sender)
	if err != nil {
		return err
	}

	if !signer.Equals(sender) {
		return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "sender %s is different from signer %s", sender, signer)
	}

	events.EmitSendCall(
		ctx,
		*data,
		sourceChannel,
		destinationChannel,
		payload.SourcePort,
		payload.DestinationPort,
		sequence,
	)

	return nil
}

func (im *IBCModule) OnRecvPacket(ctx sdk.Context, sourceClient, destinationClient string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
	var (
		ackErr error
		data   types.GMPPacketData
	)

	// we are explicitly wrapping this emit event call in an anonymous function so that
	// the packet data is evaluated after it has been assigned a value.
	defer func() {
		events.EmitOnRecvPacketEvent(
			ctx,
			data,
			sourceClient,
			destinationClient,
			payload.SourcePort,
			payload.DestinationPort,
			sequence,
			ackErr,
		)
	}()

	if payload.SourcePort != types.PortID || payload.DestinationPort != types.PortID {
		ackErr = errorsmod.Wrapf(types.ErrInvalidPayload, "payload port ID is invalid: expected %s, got sourcePort: %s destPort: %s", types.PortID, payload.SourcePort, payload.DestinationPort)
		im.keeper.Logger(ctx).Error("recv packet failed", "error", ackErr, "sequence", sequence, "destination_client", destinationClient)
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}
	if payload.Version != types.Version {
		ackErr = errorsmod.Wrapf(types.ErrInvalidVersion, "payload version is invalid: expected %s, got %s", types.Version, payload.Version)
		im.keeper.Logger(ctx).Error("recv packet failed", "error", ackErr, "sequence", sequence, "destination_client", destinationClient)
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}

	packetData, err := types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		ackErr = err
		im.keeper.Logger(ctx).Error("recv packet failed", "error", err, "sequence", sequence, "destination_client", destinationClient)
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}

	data = *packetData

	if err := packetData.ValidateBasic(); err != nil {
		ackErr = err
		im.keeper.Logger(ctx).Error("recv packet failed", "error", ackErr, "sequence", sequence, "destination_client", destinationClient)
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}

	result, err := im.keeper.OnRecvPacket(
		ctx,
		packetData,
		destinationClient,
	)
	if err != nil {
		ackErr = err
		im.keeper.Logger(ctx).Error("recv packet failed", "error", err, "sequence", sequence, "destination_client", destinationClient)
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}

	ack := types.NewAcknowledgement(result)
	ackBz, err := types.MarshalAcknowledgement(&ack, types.Version, payload.Encoding)
	if err != nil {
		ackErr = err
		im.keeper.Logger(ctx).Error("recv packet failed", "error", ackErr, "sequence", sequence, "destination_client", destinationClient)
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}

	im.keeper.Logger(ctx).Info("successfully handled ICS-27 GMP packet", "destination_client", destinationClient, "sequence", sequence)

	// TODO: implement telemetry

	return channeltypesv2.RecvPacketResult{
		Status:          channeltypesv2.PacketStatus_Success,
		Acknowledgement: ackBz,
	}
}

func (*IBCModule) OnTimeoutPacket(_ sdk.Context, _, _ string, _ uint64, _ channeltypesv2.Payload, _ sdk.AccAddress) error {
	return nil
}

func (*IBCModule) OnAcknowledgementPacket(_ sdk.Context, _, _ string, _ uint64, _ []byte, _ channeltypesv2.Payload, _ sdk.AccAddress) error {
	return nil
}

// UnmarshalPacketData unmarshals GMP packet data from the payload.
// This method implements the PacketDataUnmarshaler interface required for callbacks middleware support.
func (*IBCModule) UnmarshalPacketData(payload channeltypesv2.Payload) (any, error) {
	data, err := types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return nil, err
	}
	return data, nil
}
