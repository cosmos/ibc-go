package ratelimiting

import (
	"bytes"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/api"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

var _ api.IBCModule = (*IBCModule)(nil)

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(k keeper.Keeper) *IBCModule {
	return &IBCModule{
		keeper: k,
	}
}

// IBCModule implements the ICS26 interface for rate-limiting middleware
type IBCModule struct {
	keeper keeper.Keeper
}

// OnSendPacket implements the IBCModule interface
func (im *IBCModule) OnSendPacket(
	ctx sdk.Context,
	sourceChannel string,
	destinationChannel string,
	sequence uint64,
	payload channeltypesv2.Payload,
	signer sdk.AccAddress,
) error {
	// This will be implemented to check and enforce rate limits on outgoing packets
	return nil
}

// OnRecvPacket implements the IBCModule interface
func (im *IBCModule) OnRecvPacket(
	ctx sdk.Context,
	sourceChannel string,
	destinationChannel string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) channeltypesv2.RecvPacketResult {
	// This will be implemented to check and enforce rate limits on incoming packets

	// Default to success
	recvResult := channeltypesv2.RecvPacketResult{
		Status:          channeltypesv2.PacketStatus_Success,
		Acknowledgement: channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement(),
	}

	return recvResult
}

// OnTimeoutPacket implements the IBCModule interface
func (im *IBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	sourceChannel string,
	destinationChannel string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	// Handle timeout packet events
	return nil
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im *IBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	sourceChannel string,
	destinationChannel string,
	sequence uint64,
	acknowledgement []byte,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	// Handle acknowledgement packet events

	// Check if this is an error acknowledgement
	if bytes.Equal(acknowledgement, channeltypesv2.ErrorAcknowledgement[:]) {
		// Handle error acknowledgement
	} else {
		var ack channeltypes.Acknowledgement
		if err := types.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
			return errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal rate-limiting packet acknowledgement: %v", err)
		}
	}

	return nil
}

// UnmarshalPacketData implements the PacketDataUnmarshaler interface
func (*IBCModule) UnmarshalPacketData(payload channeltypesv2.Payload) (interface{}, error) {
	// Forward to underlying module's packet data unmarshaler if needed
	// For now, return nil as rate-limiting may just act as middleware
	return nil, nil
}
