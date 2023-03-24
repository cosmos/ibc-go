package types

import (
	"encoding/base64"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

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
	proposedUpgradeChannel Channel,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	signer string,
) *MsgChannelUpgradeInit {
	return &MsgChannelUpgradeInit{
		PortId:                 portID,
		ChannelId:              channelID,
		ProposedUpgradeChannel: proposedUpgradeChannel,
		TimeoutHeight:          timeoutHeight,
		TimeoutTimestamp:       timeoutTimestamp,
		Signer:                 signer,
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
	if msg.ProposedUpgradeChannel.State != INITUPGRADE {
		return errorsmod.Wrapf(ErrInvalidChannelState, "expected: %s, got: %s", INITUPGRADE, msg.ProposedUpgradeChannel.State)
	}
	if strings.TrimSpace(msg.ProposedUpgradeChannel.Version) == "" {
		return errorsmod.Wrap(ErrInvalidChannelVersion, "channel version must not be empty")
	}
	if msg.TimeoutHeight.IsZero() && msg.TimeoutTimestamp == 0 {
		return errorsmod.Wrap(ErrInvalidUpgradeTimeout, "timeout height and timeout timestamp cannot both be 0")
	}
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
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
	portID, channelID string,
	counterpartyChannel Channel,
	counterpartySequence uint64,
	proposedUpgradeChannel Channel,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	proofChannel []byte,
	proofUpgradeTimeout []byte,
	proofUpgradeSequence []byte,
	proofHeight clienttypes.Height,
	signer string,
) *MsgChannelUpgradeTry {
	return &MsgChannelUpgradeTry{
		PortId:                 portID,
		ChannelId:              channelID,
		CounterpartyChannel:    counterpartyChannel,
		CounterpartySequence:   counterpartySequence,
		ProposedUpgradeChannel: proposedUpgradeChannel,
		TimeoutHeight:          timeoutHeight,
		TimeoutTimestamp:       timeoutTimestamp,
		ProofChannel:           proofChannel,
		ProofUpgradeTimeout:    proofUpgradeTimeout,
		ProofUpgradeSequence:   proofUpgradeSequence,
		ProofHeight:            proofHeight,
		Signer:                 signer,
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
	if msg.CounterpartyChannel.State != INITUPGRADE {
		return errorsmod.Wrapf(ErrInvalidChannelState, "expected: %s, got: %s", INITUPGRADE, msg.CounterpartyChannel.State)
	}
	if msg.CounterpartySequence == 0 {
		return errorsmod.Wrap(ibcerrors.ErrInvalidSequence, "counterparty sequence cannot be 0")
	}
	if msg.ProposedUpgradeChannel.State != TRYUPGRADE {
		return errorsmod.Wrapf(ErrInvalidChannelState, "expected: %s, got: %s", TRYUPGRADE, msg.CounterpartyChannel.State)
	}
	if strings.TrimSpace(msg.ProposedUpgradeChannel.Version) == "" {
		return errorsmod.Wrap(ErrInvalidChannelVersion, "channel version must not be empty")
	}
	if msg.TimeoutHeight.IsZero() {
		return errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "timeout height must be non-zero")
	}
	if msg.TimeoutTimestamp == 0 && msg.TimeoutHeight.IsZero() {
		return errorsmod.Wrap(ErrInvalidUpgradeTimeout, "invalid upgrade timeout timestamp or timeout height")
	}
	if len(msg.ProofChannel) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty channel proof")
	}
	if len(msg.ProofUpgradeTimeout) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty upgrade timeout proof")
	}
	if len(msg.ProofUpgradeSequence) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty upgrade sequence proof")
	}
	if msg.ProofHeight.IsZero() {
		return errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "proof height must be non-zero")
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
func NewMsgChannelUpgradeAck(portID, channelID string, counterpartyChannel Channel, proofChannel, proofUpgradeSequence []byte, proofHeight clienttypes.Height, signer string) *MsgChannelUpgradeAck {
	return &MsgChannelUpgradeAck{
		PortId:               portID,
		ChannelId:            channelID,
		CounterpartyChannel:  counterpartyChannel,
		ProofChannel:         proofChannel,
		ProofUpgradeSequence: proofUpgradeSequence,
		ProofHeight:          proofHeight,
		Signer:               signer,
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
	if len(msg.ProofUpgradeSequence) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty upgrade sequence proof")
	}
	if msg.ProofHeight.IsZero() {
		return errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "proof height must be non-zero")
	}
	if msg.CounterpartyChannel.State != TRYUPGRADE {
		return errorsmod.Wrapf(ErrInvalidChannelState, "expected: %s, got: %s", "TRYUPGRADE", msg.CounterpartyChannel.State)
	}
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
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
// nolint:interfacer
func NewMsgChannelUpgradeConfirm(
	portID,
	channelID string,
	counterpartyChannel Channel,
	proofChannel,
	proofUpgradeError,
	proofUpgradeSequence []byte,
	proofHeight clienttypes.Height,
	signer string,
) *MsgChannelUpgradeConfirm {
	return &MsgChannelUpgradeConfirm{
		PortId:               portID,
		ChannelId:            channelID,
		CounterpartyChannel:  counterpartyChannel,
		ProofChannel:         proofChannel,
		ProofUpgradeError:    proofUpgradeError,
		ProofUpgradeSequence: proofUpgradeSequence,
		ProofHeight:          proofHeight,
		Signer:               signer,
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

	if len(msg.ProofChannel) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty channel proof")
	}

	if len(msg.ProofUpgradeError) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty upgrade error proof")
	}

	if len(msg.ProofUpgradeSequence) == 0 {
		return errorsmod.Wrap(commitmenttypes.ErrInvalidProof, "cannot submit an empty upgrade sequence proof")
	}

	if msg.ProofHeight.IsZero() {
		return errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "proof height must be non-zero")
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
func (msg MsgChannelUpgradeConfirm) GetSigners() []sdk.AccAddress {
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
	errorReceipt ErrorReceipt,
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

	if msg.ProofHeight.IsZero() {
		return errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "proof height must be non-zero")
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

	if msg.ProofHeight.IsZero() {
		return errorsmod.Wrap(ibcerrors.ErrInvalidHeight, "proof height must be non-zero")
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
