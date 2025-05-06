package gmp

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/api"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

var _ api.IBCModule = (*IBCModule)(nil)

// IBCModule implements the ICS26 interface for transfer given the transfer keeper.
type IBCModule struct {
	keeper keeper.Keeper
}

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(k keeper.Keeper) *IBCModule {
	return &IBCModule{
		keeper: k,
	}
}

func (im *IBCModule) OnSendPacket(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress) error {
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

	sender, err := sdk.AccAddressFromBech32(data.Sender)
	if err != nil {
		return err
	}

	if !signer.Equals(sender) {
		return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "sender %s is different from signer %s", sender, signer)
	}

	// TODO: emit event and telemetry

	return nil
}

func (im *IBCModule) OnRecvPacket(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
	// TODO: implement
	panic("not implemented")
}

func (im *IBCModule) OnTimeoutPacket(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	// TODO: implement
	panic("not implemented")
}

func (im *IBCModule) OnAcknowledgementPacket(ctx sdk.Context, sourceChannel string, destinationChannel string, sequence uint64, acknowledgement []byte, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	// TODO: implement
	panic("not implemented")
}
