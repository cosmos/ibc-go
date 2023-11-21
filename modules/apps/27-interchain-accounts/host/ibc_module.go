package host

import (
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var (
	_ porttypes.IBCModule             = (*IBCModule)(nil)
	_ porttypes.PacketDataUnmarshaler = (*IBCModule)(nil)
)

// IBCModule implements the ICS26 interface for interchain accounts host chains
type IBCModule struct {
	keeper keeper.Keeper
}

// NewIBCModule creates a new IBCModule given the associated keeper
func NewIBCModule(k keeper.Keeper) IBCModule {
	return IBCModule{
		keeper: k,
	}
}

// OnChanOpenInit implements the IBCModule interface
func (IBCModule) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return "", errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
}

// OnChanOpenTry implements the IBCModule interface
func (im IBCModule) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	if !im.keeper.GetParams(ctx).HostEnabled {
		return "", types.ErrHostSubModuleDisabled
	}

	return im.keeper.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCModule interface
func (IBCModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	return errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
}

// OnChanOpenAck implements the IBCModule interface
func (im IBCModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	if !im.keeper.GetParams(ctx).HostEnabled {
		return types.ErrHostSubModuleDisabled
	}

	return im.keeper.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCModule interface
func (IBCModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Disallow user-initiated channel closing for interchain account channels
	return errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCModule interface
func (im IBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.keeper.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCModule interface
func (im IBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	_ sdk.AccAddress,
) ibcexported.Acknowledgement {
	logger := im.keeper.Logger(ctx)
	if !im.keeper.GetParams(ctx).HostEnabled {
		logger.Info("host submodule is disabled")
		return channeltypes.NewErrorAcknowledgement(types.ErrHostSubModuleDisabled)
	}

	txResponse, err := im.keeper.OnRecvPacket(ctx, packet)
	ack := channeltypes.NewResultAcknowledgement(txResponse)
	if err != nil {
		ack = channeltypes.NewErrorAcknowledgement(err)
		logger.Error(fmt.Sprintf("%s sequence %d", err.Error(), packet.Sequence))
	} else {
		logger.Info("successfully handled packet sequence: %d", packet.Sequence)
	}

	// Emit an event indicating a successful or failed acknowledgement.
	keeper.EmitAcknowledgementEvent(ctx, packet, ack, err)

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return ack
}

// OnAcknowledgementPacket implements the IBCModule interface
func (IBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	return errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "cannot receive acknowledgement on a host channel end, a host chain does not send a packet over the channel")
}

// OnTimeoutPacket implements the IBCModule interface
func (IBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "cannot cause a packet timeout on a host channel end, a host chain does not send a packet over the channel")
}

// OnChanUpgradeInit implements the IBCModule interface
func (IBCModule) OnChanUpgradeInit(ctx sdk.Context, portID, channelID string, order channeltypes.Order, connectionHops []string, version string) (string, error) {
	return "", errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
}

/*
// Called on Host Chain by Relayer
function onChanUpgradeTry(
  portIdentifier: Identifier,
  channelIdentifier: Identifier,
  order: ChannelOrder,
  connectionHops: [Identifier],
  upgradeSequence: uint64,
  counterpartyPortIdentifier: Identifier,
  counterpartyChannelIdentifier: Identifier,
  counterpartyVersion: string
): (version: string, err: Error) {
  // validate port ID
  abortTransactionUnless(portIdentifier === "icahost")

  // upgrade version proposed by counterparty
  abortTransactionUnless(counterpartyVersion !== "")

  // retrieve the existing channel version.
  // In ibc-go, for example, this is done using the GetAppVersion
  // function of the ICS4Wrapper interface.
  // See https://github.com/cosmos/ibc-go/blob/ac6300bd857cd2bd6915ae51e67c92848cbfb086/modules/core/05-port/types/module.go#L128-L132
  channel = provableStore.get(channelPath(portIdentifier, channelIdentifier))
  abortTransactionUnless(channel !== null)
  currentMetadata = UnmarshalJSON(channel.version)

  // validate metadata
  abortTransactionUnless(metadata.Version === "ics27-1")
  // all elements in encoding list and tx type list must be supported
  abortTransactionUnless(IsSupportedEncoding(metadata.Encoding))
  abortTransactionUnless(IsSupportedTxType(metadata.TxType))

  // the interchain account address on the host chain
  // must remain the same after the upgrade.
  abortTransactionUnless(currentMetadata.Address === metadata.Address)

  // at the moment it is not supported to perform upgrades that
  // change the connection ID of the controller or host chains.
  // therefore these connection IDs much remain the same as before.
  abortTransactionUnless(currentMetadata.ControllerConnectionId === metadata.ControllerConnectionId)
  abortTransactionUnless(currentMetadata.HostConnectionId === metadata.HostConnectionId)
  // the proposed connection hop must not change
  abortTransactionUnless(currentMetadata.HostConnectionId === connectionHops[0])

  return counterpartyVersion, nil
}
*/
// OnChanUpgradeTry implements the IBCModule interface
func (IBCModule) OnChanUpgradeTry(ctx sdk.Context, portID, channelID string, order channeltypes.Order, connectionHops []string, counterpartyVersion string) (string, error) {
	if portID != icatypes.HostPortID {
		return "", errorsmod.Wrapf(porttypes.ErrInvalidPort, "port ID must be %s", icatypes.HostPortID)
	}

	counterpartyVersion = strings.TrimSpace(counterpartyVersion)
	if counterpartyVersion == "" {
		return "", errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "counterparty version cannot be empty")
	}

	return icatypes.Version, nil
}

// OnChanUpgradeAck implements the IBCModule interface
func (IBCModule) OnChanUpgradeAck(ctx sdk.Context, portID, channelID, counterpartyVersion string) error {
	return errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
}

// OnChanUpgradeOpen implements the IBCModule interface
func (IBCModule) OnChanUpgradeOpen(ctx sdk.Context, portID, channelID string, order channeltypes.Order, connectionHops []string, version string) {
}

// OnChanUpgradeRestore implements the IBCModule interface
func (IBCModule) OnChanUpgradeRestore(ctx sdk.Context, portID, channelID string) {}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes
// into an InterchainAccountPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (IBCModule) UnmarshalPacketData(bz []byte) (interface{}, error) {
	var data icatypes.InterchainAccountPacketData
	err := data.UnmarshalJSON(bz)
	if err != nil {
		return nil, err
	}
	return data, nil
}
