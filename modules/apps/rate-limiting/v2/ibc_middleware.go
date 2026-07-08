package v2

import (
	"encoding/json"
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/keeper"
	ratelimitingtypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v11/modules/core/api"
)

var (
	_ api.IBCModule                   = (*IBCMiddleware)(nil)
	_ api.WriteAcknowledgementWrapper = (*IBCMiddleware)(nil)
)

type IBCMiddleware struct {
	app             api.IBCModule
	keeper          keeper.Keeper
	writeAckWrapper api.WriteAcknowledgementWrapper
	chanKeeperV2    ratelimitingtypes.ChannelKeeperV2
}

func NewIBCMiddleware(k keeper.Keeper, app api.IBCModule, writeAckWrapper api.WriteAcknowledgementWrapper, chanKeeperV2 ratelimitingtypes.ChannelKeeperV2) IBCMiddleware {
	if writeAckWrapper == nil {
		panic(errors.New("write acknowledgement wrapper cannot be nil"))
	}

	if chanKeeperV2 == nil {
		panic(errors.New("channel keeper v2 cannot be nil"))
	}

	return IBCMiddleware{
		app:             app,
		keeper:          k,
		writeAckWrapper: writeAckWrapper,
		chanKeeperV2:    chanKeeperV2,
	}
}

func (im IBCMiddleware) OnSendPacket(ctx sdk.Context, sourceClient string, destinationClient string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress) error {
	packet, err := v2ToV1Packet(payload, sourceClient, destinationClient, sequence)
	if err != nil {
		im.keeper.Logger(ctx).Error("ICS20 rate limiting OnSendPacket failed to convert v2 packet to v1 packet", "error", err)
		return err
	}
	if err := im.keeper.SendRateLimitedPacket(ctx, packet.SourcePort, packet.SourceChannel, packet.TimeoutHeight, packet.TimeoutTimestamp, packet.Data); err != nil {
		im.keeper.Logger(ctx).Error("ICS20 packet send was denied", "error", err)
		return err
	}
	return im.app.OnSendPacket(ctx, sourceClient, destinationClient, sequence, payload, signer)
}

func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, sourceClient string, destinationClient string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
	packet, err := v2ToV1Packet(payload, sourceClient, destinationClient, sequence)
	if err != nil {
		im.keeper.Logger(ctx).Error("ICS20 rate limiting OnRecvPacket failed to convert v2 packet to v1 packet", "error", err)
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: channeltypes.NewErrorAcknowledgement(err).Acknowledgement(),
		}
	}
	// Check if the packet would cause the rate limit to be exceeded,
	// and if so, return an ack error
	if err := im.keeper.ReceiveRateLimitedPacket(ctx, packet); err != nil {
		im.keeper.Logger(ctx).Error("ICS20 packet receive was denied", "error", err)
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: channeltypes.NewErrorAcknowledgement(err).Acknowledgement(),
		}
	}

	// If the acknowledgement is not async, remove the pending receive packet from the store.
	result := im.app.OnRecvPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
	if result.Status != channeltypesv2.PacketStatus_Async {
		packetInfo, err := keeper.ParsePacketInfo(packet, ratelimitingtypes.PACKET_RECV)
		if err != nil {
			im.keeper.Logger(ctx).Error("ICS20 rate limiting OnRecvPacket failed to parse packet data for pending receive cleanup", "error", err)
			return result
		}
		if err := im.keeper.RemovePendingReceivePacket(ctx, packetInfo.ChannelID, sequence, packetInfo.Denom); err != nil {
			im.keeper.Logger(ctx).Error("ICS20 rate limiting OnRecvPacket failed to remove pending receive packet", "error", err)
		}
	}

	return result
}

func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, sourceClient string, destinationClient string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	packet, err := v2ToV1Packet(payload, sourceClient, destinationClient, sequence)
	if err != nil {
		im.keeper.Logger(ctx).Error("ICS20 rate limiting OnTimeoutPacket failed to convert v2 packet to v1 packet", "error", err)
		return err
	}
	if err := im.keeper.TimeoutRateLimitedPacket(ctx, packet); err != nil {
		im.keeper.Logger(ctx).Error("ICS20 RateLimited OnTimeoutPacket failed", "error", err)
		return err
	}
	return im.app.OnTimeoutPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
}

func (im IBCMiddleware) OnAcknowledgementPacket(ctx sdk.Context, sourceClient string, destinationClient string, sequence uint64, acknowledgement []byte, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	packet, err := v2ToV1Packet(payload, sourceClient, destinationClient, sequence)
	if err != nil {
		im.keeper.Logger(ctx).Error("ICS20 rate limiting OnAckPacketfailed to convert v2 packet to v1 packet", "error", err)
		return err
	}
	if err := im.keeper.AcknowledgeRateLimitedPacket(ctx, packet, acknowledgement); err != nil {
		im.keeper.Logger(ctx).Error("ICS20 RateLimited OnAckPacket failed", "error", err)
		return err
	}
	return im.app.OnAcknowledgementPacket(ctx, sourceClient, destinationClient, sequence, acknowledgement, payload, relayer)
}

func (im IBCMiddleware) WriteAcknowledgement(ctx sdk.Context, clientID string, sequence uint64, ack channeltypesv2.Acknowledgement) error {
	packet, found := im.chanKeeperV2.GetAsyncPacket(ctx, clientID, sequence)
	if !found {
		im.keeper.Logger(ctx).Error("ICS20 rate limiting WriteAcknowledgement failed: async packet not found", "clientID", clientID, "sequence", sequence)
		return ratelimitingtypes.ErrAsyncPacketNotFound.Wrapf("clientID: %s, sequence: %d", clientID, sequence)
	}

	// NOTE: async acknowledgements can only be for single payload packets
	if len(ack.AppAcknowledgements) != 1 || len(packet.Payloads) != 1 {
		im.keeper.Logger(ctx).Error("ICS20 rate limiting WriteAcknowledgement failed: async acknowledgements can only be for single payload packets", "clientID", clientID, "sequence", sequence)
		return im.writeAckWrapper.WriteAcknowledgement(ctx, clientID, sequence, ack)
	}

	v1Packet, err := v2ToV1Packet(packet.Payloads[0], packet.SourceClient, packet.DestinationClient, packet.Sequence)
	if err != nil {
		im.keeper.Logger(ctx).Error("ICS20 rate limiting WriteAcknowledgement failed to convert v2 packet to v1 packet", "error", err)
		return err
	}

	if ack.Success() {
		packetInfo, err := keeper.ParsePacketInfo(v1Packet, ratelimitingtypes.PACKET_RECV)
		if err != nil {
			im.keeper.Logger(ctx).Error("ICS20 rate limiting WriteAcknowledgement failed to parse packet data for pending receive cleanup", "error", err)
			return im.writeAckWrapper.WriteAcknowledgement(ctx, clientID, sequence, ack)
		}
		if err := im.keeper.RemovePendingReceivePacket(ctx, packetInfo.ChannelID, sequence, packetInfo.Denom); err != nil {
			im.keeper.Logger(ctx).Error("ICS20 rate limiting WriteAcknowledgement failed to remove pending receive packet", "error", err)
			return err
		}
	} else {
		if err := im.keeper.UndoReceivePacket(ctx, v1Packet); err != nil {
			im.keeper.Logger(ctx).Error("ICS20 rate limiting WriteAcknowledgement failed to undo receive packet", "error", err)
			return err
		}
	}

	return im.writeAckWrapper.WriteAcknowledgement(ctx, clientID, sequence, ack)
}

// TODO: Something looks off about this, please review carefully
func v2ToV1Packet(payload channeltypesv2.Payload, sourceClient, destinationClient string, sequence uint64) (channeltypes.Packet, error) {
	transferRepresentation, err := transfertypes.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return channeltypes.Packet{}, err
	}

	packetData := transfertypes.FungibleTokenPacketData{
		Denom:    transferRepresentation.Token.Denom.Path(),
		Amount:   transferRepresentation.Token.Amount,
		Sender:   transferRepresentation.Sender,
		Receiver: transferRepresentation.Receiver,
		Memo:     transferRepresentation.Memo,
	}

	packetDataBz, err := json.Marshal(packetData)
	if err != nil {
		return channeltypes.Packet{}, err
	}

	return channeltypes.Packet{
		Sequence:           sequence,
		SourcePort:         payload.SourcePort,
		SourceChannel:      sourceClient,
		DestinationPort:    payload.DestinationPort,
		DestinationChannel: destinationClient,
		Data:               packetDataBz,
		TimeoutHeight:      clienttypes.Height{},
		TimeoutTimestamp:   0,
	}, nil
}
