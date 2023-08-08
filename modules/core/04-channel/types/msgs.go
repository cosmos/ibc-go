package types

import (
	"encoding/base64"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/internal/collections"
	ibcerrors "github.com/cosmos/ibc-go/v7/internal/errors"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
)

var _ sdk.Msg = &MsgChannelOpenInit{}

// NewMsgChannelOpenInit creates a new MsgChannelOpenInit. It sets the counterparty channel
// identifier to be empty.
//
//nolint:interfacer
func NewMsgChannelOpenInit(
	portID, version string, channelOrder Order, connectionHops []string,
	counterpartyPortID string, signer string,
) *MsgChannelOpenInit {
	counterparty := NewCounterparty(counterpartyPortID, "")
	channel := NewChannel(INIT, channelOrder, counterparty, connectionHops, version)
	return &MsgChannelOpenInit{
		PortId:  portID,
		Channel: channel,
		Signer:  signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelOpenInit) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}
	if msg.Channel.State != INIT {
		return errorsmod.Wrapf(ErrInvalidChannelState,
			"channel state must be INIT in MsgChannelOpenInit. expected: %s, got: %s",
			INIT, msg.Channel.State,
		)
	}
	if msg.Channel.Counterparty.ChannelId != "" {
		return errorsmod.Wrap(ErrInvalidCounterparty, "counterparty channel identifier must be empty")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return msg.Channel.ValidateBasic()
}

// GetSigners implements sdk.Msg
func (msg MsgChannelOpenInit) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelOpenTry{}

// NewMsgChannelOpenTry creates a new MsgChannelOpenTry instance
// The version string is deprecated and will be ignored by core IBC.
// It is left as an argument for go API backwards compatibility.
//
//nolint:interfacer
func NewMsgChannelOpenTry(
	portID, version string, channelOrder Order, connectionHops []string,
	counterpartyPortID, counterpartyChannelID, counterpartyVersion string,
	proofInit []byte, proofHeight clienttypes.Height, signer string,
) *MsgChannelOpenTry {
	counterparty := NewCounterparty(counterpartyPortID, counterpartyChannelID)
	channel := NewChannel(TRYOPEN, channelOrder, counterparty, connectionHops, version)
	return &MsgChannelOpenTry{
		PortId:              portID,
		Channel:             channel,
		CounterpartyVersion: counterpartyVersion,
		ProofInit:           proofInit,
		ProofHeight:         proofHeight,
		Signer:              signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelOpenTry) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}
	if msg.PreviousChannelId != "" {
		return errorsmod.Wrap(ErrInvalidChannelIdentifier, "previous channel identifier must be empty, this field has been deprecated as crossing hellos are no longer supported")
	}
	if len(msg.ProofInit) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty init proof")
	}
	if msg.Channel.State != TRYOPEN {
		return errorsmod.Wrapf(ErrInvalidChannelState,
			"channel state must be TRYOPEN in MsgChannelOpenTry. expected: %s, got: %s",
			TRYOPEN, msg.Channel.State,
		)
	}
	// counterparty validate basic allows empty counterparty channel identifiers
	if err := host.ChannelIdentifierValidator(msg.Channel.Counterparty.ChannelId); err != nil {
		return errorsmod.Wrap(err, "invalid counterparty channel ID")
	}

	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return msg.Channel.ValidateBasic()
}

// GetSigners implements sdk.Msg
func (msg MsgChannelOpenTry) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelOpenAck{}

// NewMsgChannelOpenAck creates a new MsgChannelOpenAck instance
//
//nolint:interfacer
func NewMsgChannelOpenAck(
	portID, channelID, counterpartyChannelID string, cpv string, proofTry []byte, proofHeight clienttypes.Height,
	signer string,
) *MsgChannelOpenAck {
	return &MsgChannelOpenAck{
		PortId:                portID,
		ChannelId:             channelID,
		CounterpartyChannelId: counterpartyChannelID,
		CounterpartyVersion:   cpv,
		ProofTry:              proofTry,
		ProofHeight:           proofHeight,
		Signer:                signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelOpenAck) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}
	if !IsValidChannelID(msg.ChannelId) {
		return ErrInvalidChannelIdentifier
	}
	if err := host.ChannelIdentifierValidator(msg.CounterpartyChannelId); err != nil {
		return errorsmod.Wrap(err, "invalid counterparty channel ID")
	}
	if len(msg.ProofTry) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty try proof")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgChannelOpenAck) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelOpenConfirm{}

// NewMsgChannelOpenConfirm creates a new MsgChannelOpenConfirm instance
//
//nolint:interfacer
func NewMsgChannelOpenConfirm(
	portID, channelID string, proofAck []byte, proofHeight clienttypes.Height,
	signer string,
) *MsgChannelOpenConfirm {
	return &MsgChannelOpenConfirm{
		PortId:      portID,
		ChannelId:   channelID,
		ProofAck:    proofAck,
		ProofHeight: proofHeight,
		Signer:      signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelOpenConfirm) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}
	if !IsValidChannelID(msg.ChannelId) {
		return ErrInvalidChannelIdentifier
	}
	if len(msg.ProofAck) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty acknowledgement proof")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgChannelOpenConfirm) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelCloseInit{}

// NewMsgChannelCloseInit creates a new MsgChannelCloseInit instance
//
//nolint:interfacer
func NewMsgChannelCloseInit(
	portID string, channelID string, signer string,
) *MsgChannelCloseInit {
	return &MsgChannelCloseInit{
		PortId:    portID,
		ChannelId: channelID,
		Signer:    signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelCloseInit) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}
	if !IsValidChannelID(msg.ChannelId) {
		return ErrInvalidChannelIdentifier
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgChannelCloseInit) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelCloseConfirm{}

// NewMsgChannelCloseConfirm creates a new MsgChannelCloseConfirm instance
//
//nolint:interfacer
func NewMsgChannelCloseConfirm(
	portID, channelID string, proofInit []byte, proofHeight clienttypes.Height,
	signer string,
) *MsgChannelCloseConfirm {
	return &MsgChannelCloseConfirm{
		PortId:      portID,
		ChannelId:   channelID,
		ProofInit:   proofInit,
		ProofHeight: proofHeight,
		Signer:      signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelCloseConfirm) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}
	if !IsValidChannelID(msg.ChannelId) {
		return ErrInvalidChannelIdentifier
	}
	if len(msg.ProofInit) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty init proof")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgChannelCloseConfirm) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgRecvPacket{}

// NewMsgRecvPacket constructs new MsgRecvPacket
//
//nolint:interfacer
func NewMsgRecvPacket(
	packet Packet, proofCommitment []byte, proofHeight clienttypes.Height,
	signer string,
) *MsgRecvPacket {
	return &MsgRecvPacket{
		Packet:          packet,
		ProofCommitment: proofCommitment,
		ProofHeight:     proofHeight,
		Signer:          signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgRecvPacket) ValidateBasic() error {
	if len(msg.ProofCommitment) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty commitment proof")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return msg.Packet.ValidateBasic()
}

// GetDataSignBytes returns the base64-encoded bytes used for the
// data field when signing the packet.
func (msg MsgRecvPacket) GetDataSignBytes() []byte {
	s := "\"" + base64.StdEncoding.EncodeToString(msg.Packet.Data) + "\""
	return []byte(s)
}

// GetSigners implements sdk.Msg
func (msg MsgRecvPacket) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgTimeout{}

// NewMsgTimeout constructs new MsgTimeout
//
//nolint:interfacer
func NewMsgTimeout(
	packet Packet, nextSequenceRecv uint64, proofUnreceived []byte,
	proofHeight clienttypes.Height, signer string,
) *MsgTimeout {
	return &MsgTimeout{
		Packet:           packet,
		NextSequenceRecv: nextSequenceRecv,
		ProofUnreceived:  proofUnreceived,
		ProofHeight:      proofHeight,
		Signer:           signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgTimeout) ValidateBasic() error {
	if len(msg.ProofUnreceived) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty unreceived proof")
	}
	if msg.NextSequenceRecv == 0 {
		return errorsmod.Wrap(ibcerrors.ErrInvalidSequence, "next sequence receive cannot be 0")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return msg.Packet.ValidateBasic()
}

// GetSigners implements sdk.Msg
func (msg MsgTimeout) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// NewMsgTimeoutOnClose constructs new MsgTimeoutOnClose
//
//nolint:interfacer
func NewMsgTimeoutOnClose(
	packet Packet, nextSequenceRecv uint64,
	proofUnreceived, proofClose []byte,
	proofHeight clienttypes.Height, signer string,
) *MsgTimeoutOnClose {
	return &MsgTimeoutOnClose{
		Packet:           packet,
		NextSequenceRecv: nextSequenceRecv,
		ProofUnreceived:  proofUnreceived,
		ProofClose:       proofClose,
		ProofHeight:      proofHeight,
		Signer:           signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgTimeoutOnClose) ValidateBasic() error {
	if msg.NextSequenceRecv == 0 {
		return errorsmod.Wrap(ibcerrors.ErrInvalidSequence, "next sequence receive cannot be 0")
	}
	if len(msg.ProofUnreceived) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty unreceived proof")
	}
	if len(msg.ProofClose) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty proof of closed counterparty channel end")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return msg.Packet.ValidateBasic()
}

// GetSigners implements sdk.Msg
func (msg MsgTimeoutOnClose) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgAcknowledgement{}

// NewMsgAcknowledgement constructs a new MsgAcknowledgement
//
//nolint:interfacer
func NewMsgAcknowledgement(
	packet Packet,
	ack, proofAcked []byte,
	proofHeight clienttypes.Height,
	signer string,
) *MsgAcknowledgement {
	return &MsgAcknowledgement{
		Packet:          packet,
		Acknowledgement: ack,
		ProofAcked:      proofAcked,
		ProofHeight:     proofHeight,
		Signer:          signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgAcknowledgement) ValidateBasic() error {
	if len(msg.ProofAcked) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty acknowledgement proof")
	}
	if len(msg.Acknowledgement) == 0 {
		return errorsmod.Wrap(ErrInvalidAcknowledgement, "ack bytes cannot be empty")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}
	return msg.Packet.ValidateBasic()
}

// GetSigners implements sdk.Msg
func (msg MsgAcknowledgement) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelUpgradeInit{}

// NewMsgChannelUpgradeInit constructs a new MsgChannelUpgradeInit
// nolint:interfacer
func NewMsgChannelUpgradeInit(
	portID, channelID string,
	upgradeFields UpgradeFields,
	upgradeTimeout Timeout,
	signer string,
) *MsgChannelUpgradeInit {
	return &MsgChannelUpgradeInit{
		PortId:    portID,
		ChannelId: channelID,
		Fields:    upgradeFields,
		Timeout:   upgradeTimeout,
		Signer:    signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelUpgradeInit) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}
	if !IsValidChannelID(msg.ChannelId) {
		return ErrInvalidChannelIdentifier
	}

	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	if !msg.Timeout.IsValid() {
		return errorsmod.Wrap(ErrInvalidUpgrade, "upgrade timeout height and upgrade timeout timestamp cannot both be 0")
	}

	return msg.Fields.ValidateBasic()
}

// GetSigners implements sdk.Msg
func (msg MsgChannelUpgradeInit) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelUpgradeTry{}

// NewMsgChannelUpgradeTry constructs a new MsgChannelUpgradeTry
// nolint:interfacer
func NewMsgChannelUpgradeTry(
	portID,
	channelID string,
	proposedConnectionHops []string,
	upgradeTimeout Timeout,
	counterpartyProposedUpgrade Upgrade,
	counterpartyUpgradeSequence uint64,
	proofChannel []byte,
	proofUpgrade []byte,
	proofHeight clienttypes.Height,
	signer string,
) *MsgChannelUpgradeTry {
	return &MsgChannelUpgradeTry{
		PortId:                        portID,
		ChannelId:                     channelID,
		ProposedUpgradeConnectionHops: proposedConnectionHops,
		UpgradeTimeout:                upgradeTimeout,
		CounterpartyProposedUpgrade:   counterpartyProposedUpgrade,
		CounterpartyUpgradeSequence:   counterpartyUpgradeSequence,
		ProofChannel:                  proofChannel,
		ProofUpgrade:                  proofUpgrade,
		ProofHeight:                   proofHeight,
		Signer:                        signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelUpgradeTry) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}

	if !IsValidChannelID(msg.ChannelId) {
		return ErrInvalidChannelIdentifier
	}

	if len(msg.ProposedUpgradeConnectionHops) == 0 {
		return errorsmod.Wrap(ErrInvalidUpgrade, "proposed connection hops cannot be empty")
	}

	if !msg.UpgradeTimeout.IsValid() {
		return errorsmod.Wrap(ErrInvalidTimeout, "upgrade timeout height or timeout timestamp must be non-zero")
	}

	if err := msg.CounterpartyProposedUpgrade.ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "error validating counterparty upgrade")
	}

	if msg.CounterpartyUpgradeSequence == 0 {
		return errorsmod.Wrap(ErrInvalidUpgradeSequence, "counterparty sequence cannot be 0")
	}

	if len(msg.ProofChannel) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty channel proof")
	}

	if len(msg.ProofUpgrade) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty upgrade proof")
	}

	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgChannelUpgradeTry) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelUpgradeAck{}

// NewMsgChannelUpgradeAck constructs a new MsgChannelUpgradeAck
// nolint:interfacer
func NewMsgChannelUpgradeAck(portID, channelID string, counterpartyFlushStatus FlushStatus, counterpartyUpgrade Upgrade, proofChannel, proofUpgrade []byte, proofHeight clienttypes.Height, signer string) *MsgChannelUpgradeAck {
	return &MsgChannelUpgradeAck{
		PortId:                  portID,
		ChannelId:               channelID,
		CounterpartyFlushStatus: counterpartyFlushStatus,
		CounterpartyUpgrade:     counterpartyUpgrade,
		ProofChannel:            proofChannel,
		ProofUpgrade:            proofUpgrade,
		ProofHeight:             proofHeight,
		Signer:                  signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelUpgradeAck) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}
	if !IsValidChannelID(msg.ChannelId) {
		return ErrInvalidChannelIdentifier
	}
	if len(msg.ProofChannel) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty channel proof")
	}
	if len(msg.ProofUpgrade) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty upgrade sequence proof")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return msg.CounterpartyUpgrade.ValidateBasic()
}

// GetSigners implements sdk.Msg
func (msg MsgChannelUpgradeAck) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelUpgradeConfirm{}

// NewMsgChannelUpgradeConfirm constructs a new MsgChannelUpgradeConfirm
func NewMsgChannelUpgradeConfirm(
	portID,
	channelID string,
	counterpartyChannelState State,
	counterpartyUpgrade Upgrade,
	proofChannel,
	proofUpgrade []byte,
	proofHeight clienttypes.Height,
	signer string,
) *MsgChannelUpgradeConfirm {
	return &MsgChannelUpgradeConfirm{
		PortId:                   portID,
		ChannelId:                channelID,
		CounterpartyChannelState: counterpartyChannelState,
		CounterpartyUpgrade:      counterpartyUpgrade,
		ProofChannel:             proofChannel,
		ProofUpgrade:             proofUpgrade,
		ProofHeight:              proofHeight,
		Signer:                   signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelUpgradeConfirm) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}

	if !IsValidChannelID(msg.ChannelId) {
		return ErrInvalidChannelIdentifier
	}

	if !collections.Contains(msg.CounterpartyChannelState, []State{STATE_FLUSHING, STATE_FLUSHCOMPLETE}) {
		return errorsmod.Wrapf(ErrInvalidChannelState, "expected channel state to be one of: %s or %s, got: %s", STATE_FLUSHING, STATE_FLUSHCOMPLETE, msg.CounterpartyChannelState)
	}

	if len(msg.ProofChannel) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty channel proof")
	}

	if len(msg.ProofUpgrade) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty upgrade proof")
	}

	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return msg.CounterpartyUpgrade.ValidateBasic()
}

// GetSigners implements sdk.Msg
func (msg MsgChannelUpgradeConfirm) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelUpgradeOpen{}

// NewMsgChannelUpgradeOpen constructs a new MsgChannelUpgradeOpen
// nolint:interfacer
func NewMsgChannelUpgradeOpen(
	portID,
	channelID string,
	counterpartyChannelState State,
	proofChannel []byte,
	proofHeight clienttypes.Height,
	signer string,
) *MsgChannelUpgradeOpen {
	return &MsgChannelUpgradeOpen{
		PortId:                   portID,
		ChannelId:                channelID,
		CounterpartyChannelState: counterpartyChannelState,
		ProofChannel:             proofChannel,
		ProofHeight:              proofHeight,
		Signer:                   signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelUpgradeOpen) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}

	if !IsValidChannelID(msg.ChannelId) {
		return ErrInvalidChannelIdentifier
	}

	if len(msg.ProofChannel) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty channel proof")
	}

	if !collections.Contains(msg.CounterpartyChannelState, []State{TRYUPGRADE, ACKUPGRADE, OPEN}) {
		return errorsmod.Wrapf(ErrInvalidChannelState, "expected channel state to be one of: %s, %s or %s, got: %s", TRYUPGRADE, ACKUPGRADE, OPEN, msg.CounterpartyChannelState)
	}

	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgChannelUpgradeOpen) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelUpgradeTimeout{}

// NewMsgChannelUpgradeTimeout constructs a new MsgChannelUpgradeTimeout
// nolint:interfacer
func NewMsgChannelUpgradeTimeout(
	portID, channelID string,
	counterpartyChannel Channel,
	errorReceipt *ErrorReceipt,
	proofChannel, proofErrorReceipt []byte,
	proofHeight clienttypes.Height,
	signer string,
) *MsgChannelUpgradeTimeout {
	return &MsgChannelUpgradeTimeout{
		PortId:               portID,
		ChannelId:            channelID,
		CounterpartyChannel:  counterpartyChannel,
		PreviousErrorReceipt: errorReceipt,
		ProofChannel:         proofChannel,
		ProofErrorReceipt:    proofErrorReceipt,
		ProofHeight:          proofHeight,
		Signer:               signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelUpgradeTimeout) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}

	if !IsValidChannelID(msg.ChannelId) {
		return ErrInvalidChannelIdentifier
	}

	if len(msg.ProofChannel) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty proof")
	}

	if msg.CounterpartyChannel.State != OPEN {
		return errorsmod.Wrapf(ErrInvalidChannelState, "expected: %s, got: %s", OPEN, msg.CounterpartyChannel.State)
	}

	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgChannelUpgradeTimeout) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}

var _ sdk.Msg = &MsgChannelUpgradeCancel{}

// NewMsgChannelUpgradeCancel constructs a new MsgChannelUpgradeCancel
// nolint:interfacer
func NewMsgChannelUpgradeCancel(
	portID, channelID string,
	errorReceipt ErrorReceipt,
	proofErrReceipt []byte,
	proofHeight clienttypes.Height,
	signer string,
) *MsgChannelUpgradeCancel {
	return &MsgChannelUpgradeCancel{
		PortId:            portID,
		ChannelId:         channelID,
		ErrorReceipt:      errorReceipt,
		ProofErrorReceipt: proofErrReceipt,
		ProofHeight:       proofHeight,
		Signer:            signer,
	}
}

// ValidateBasic implements sdk.Msg
func (msg MsgChannelUpgradeCancel) ValidateBasic() error {
	if err := host.PortIdentifierValidator(msg.PortId); err != nil {
		return errorsmod.Wrap(err, "invalid port ID")
	}

	if !IsValidChannelID(msg.ChannelId) {
		return ErrInvalidChannelIdentifier
	}

	if len(msg.ProofErrorReceipt) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty proof")
	}

	if msg.ErrorReceipt.Sequence == 0 {
		return errorsmod.Wrap(ibcerrors.ErrInvalidSequence, "upgrade sequence cannot be 0")
	}

	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgChannelUpgradeCancel) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{signer}
}
