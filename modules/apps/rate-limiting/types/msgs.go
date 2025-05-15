package types

import (
	"fmt"
	"regexp"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

const (
	TypeMsgAddRateLimit    = "AddRateLimit"
	TypeMsgUpdateRateLimit = "UpdateRateLimit"
	TypeMsgRemoveRateLimit = "RemoveRateLimit"
	TypeMsgResetRateLimit  = "ResetRateLimit"
)

var (
	_ sdk.Msg = &MsgAddRateLimit{}
	_ sdk.Msg = &MsgUpdateRateLimit{}
	_ sdk.Msg = &MsgRemoveRateLimit{}
	_ sdk.Msg = &MsgResetRateLimit{}
	_ sdk.Msg = &MsgUpdateParams{}

	// Implement legacy interface for ledger support
	_ legacytx.LegacyMsg = &MsgAddRateLimit{}
	_ legacytx.LegacyMsg = &MsgUpdateRateLimit{}
	_ legacytx.LegacyMsg = &MsgRemoveRateLimit{}
	_ legacytx.LegacyMsg = &MsgResetRateLimit{}
	_ legacytx.LegacyMsg = &MsgUpdateParams{}
)

// ----------------------------------------------
//               MsgAddRateLimit
// ----------------------------------------------

func NewMsgAddRateLimit(denom, channelOrClientId string, maxPercentSend sdkmath.Int, maxPercentRecv sdkmath.Int, durationHours uint64) *MsgAddRateLimit {
	return &MsgAddRateLimit{
		Denom:             denom,
		ChannelOrClientId: channelOrClientId,
		MaxPercentSend:    maxPercentSend,
		MaxPercentRecv:    maxPercentRecv,
		DurationHours:     durationHours,
	}
}

func (msg MsgAddRateLimit) Type() string {
	return TypeMsgAddRateLimit
}

func (msg MsgAddRateLimit) Route() string {
	return RouterKey
}

func (msg *MsgAddRateLimit) GetSigners() []sdk.AccAddress {
	signerAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signerAddr}
}

func (msg *MsgAddRateLimit) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddRateLimit) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid denom (%s)", msg.Denom)
	}

	matched, err := regexp.MatchString(`^channel-\d+$`, msg.ChannelOrClientId)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "unable to verify channel-id (%s)", msg.ChannelOrClientId)
	}
	if !matched && !clienttypes.IsValidClientID(msg.ChannelOrClientId) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"invalid channel or client-id (%s), must be of the format 'channel-{N}' or a valid client-id", msg.ChannelOrClientId)
	}

	if msg.MaxPercentSend.GT(sdkmath.NewInt(100)) || msg.MaxPercentSend.LT(sdkmath.ZeroInt()) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"max-percent-send percent must be between 0 and 100 (inclusively), Provided: %v", msg.MaxPercentSend)
	}

	if msg.MaxPercentRecv.GT(sdkmath.NewInt(100)) || msg.MaxPercentRecv.LT(sdkmath.ZeroInt()) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"max-percent-recv percent must be between 0 and 100 (inclusively), Provided: %v", msg.MaxPercentRecv)
	}

	if msg.MaxPercentRecv.IsZero() && msg.MaxPercentSend.IsZero() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"either the max send or max receive threshold must be greater than 0")
	}

	if msg.DurationHours == 0 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "duration can not be zero")
	}

	return nil
}

// ----------------------------------------------
//               MsgUpdateRateLimit
// ----------------------------------------------

func NewMsgUpdateRateLimit(denom, channelOrClientId string, maxPercentSend sdkmath.Int, maxPercentRecv sdkmath.Int, durationHours uint64) *MsgUpdateRateLimit {
	return &MsgUpdateRateLimit{
		Denom:             denom,
		ChannelOrClientId: channelOrClientId,
		MaxPercentSend:    maxPercentSend,
		MaxPercentRecv:    maxPercentRecv,
		DurationHours:     durationHours,
	}
}

func (msg MsgUpdateRateLimit) Type() string {
	return TypeMsgUpdateRateLimit
}

func (msg MsgUpdateRateLimit) Route() string {
	return RouterKey
}

func (msg *MsgUpdateRateLimit) GetSigners() []sdk.AccAddress {
	signerAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signerAddr}
}

func (msg *MsgUpdateRateLimit) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUpdateRateLimit) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid denom (%s)", msg.Denom)
	}

	matched, err := regexp.MatchString(`^channel-\d+$`, msg.ChannelOrClientId)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "unable to verify channel-id (%s)", msg.ChannelOrClientId)
	}
	if !matched && !clienttypes.IsValidClientID(msg.ChannelOrClientId) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"invalid channel or client-id (%s), must be of the format 'channel-{N}' or a valid client-id", msg.ChannelOrClientId)
	}

	if msg.MaxPercentSend.GT(sdkmath.NewInt(100)) || msg.MaxPercentSend.LT(sdkmath.ZeroInt()) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"max-percent-send percent must be between 0 and 100 (inclusively), Provided: %v", msg.MaxPercentSend)
	}

	if msg.MaxPercentRecv.GT(sdkmath.NewInt(100)) || msg.MaxPercentRecv.LT(sdkmath.ZeroInt()) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"max-percent-recv percent must be between 0 and 100 (inclusively), Provided: %v", msg.MaxPercentRecv)
	}

	if msg.MaxPercentRecv.IsZero() && msg.MaxPercentSend.IsZero() {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"either the max send or max receive threshold must be greater than 0")
	}

	if msg.DurationHours == 0 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "duration can not be zero")
	}

	return nil
}

// ----------------------------------------------
//               MsgRemoveRateLimit
// ----------------------------------------------

func NewMsgRemoveRateLimit(denom, channelOrClientId string) *MsgRemoveRateLimit {
	return &MsgRemoveRateLimit{
		Denom:             denom,
		ChannelOrClientId: channelOrClientId,
	}
}

func (msg MsgRemoveRateLimit) Type() string {
	return TypeMsgRemoveRateLimit
}

func (msg MsgRemoveRateLimit) Route() string {
	return RouterKey
}

func (msg *MsgRemoveRateLimit) GetSigners() []sdk.AccAddress {
	signerAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signerAddr}
}

func (msg *MsgRemoveRateLimit) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRemoveRateLimit) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid denom (%s)", msg.Denom)
	}

	matched, err := regexp.MatchString(`^channel-\d+$`, msg.ChannelOrClientId)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "unable to verify channel-id (%s)", msg.ChannelOrClientId)
	}
	if !matched && !clienttypes.IsValidClientID(msg.ChannelOrClientId) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"invalid channel or client-id (%s), must be of the format 'channel-{N}' or a valid client-id", msg.ChannelOrClientId)
	}

	return nil
}

// ----------------------------------------------
//               MsgResetRateLimit
// ----------------------------------------------

func NewMsgResetRateLimit(denom, channelOrClientId string) *MsgResetRateLimit {
	return &MsgResetRateLimit{
		Denom:             denom,
		ChannelOrClientId: channelOrClientId,
	}
}

func (msg MsgResetRateLimit) Type() string {
	return TypeMsgResetRateLimit
}

func (msg MsgResetRateLimit) Route() string {
	return RouterKey
}

func (msg *MsgResetRateLimit) GetSigners() []sdk.AccAddress {
	signerAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signerAddr}
}

func (msg *MsgResetRateLimit) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgResetRateLimit) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}

	if msg.Denom == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid denom (%s)", msg.Denom)
	}

	matched, err := regexp.MatchString(`^channel-\d+$`, msg.ChannelOrClientId)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "unable to verify channel-id (%s)", msg.ChannelOrClientId)
	}
	if !matched && !clienttypes.IsValidClientID(msg.ChannelOrClientId) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"invalid channel or client-id (%s), must be of the format 'channel-{N}' or a valid client-id", msg.ChannelOrClientId)
	}

	return nil
}

// ----------------------------------------------
//               MsgUpdateParams
// ----------------------------------------------

// TypeMsgUpdateParams defines the type for MsgUpdateParams
const TypeMsgUpdateParams = "UpdateParams"

// MsgUpdateParams defines the Msg/UpdateParams request type
type MsgUpdateParams struct {
	// Signer is the governance account that signs the message
	Signer string `protobuf:"bytes,1,opt,name=signer,proto3" json:"signer,omitempty"`
	// Params defines the rate-limiting parameters to update
	Params Params `protobuf:"bytes,2,opt,name=params,proto3" json:"params"`
}

// Reset implements proto.Message
func (msg *MsgUpdateParams) Reset() {
	*msg = MsgUpdateParams{}
}

// String implements proto.Message
func (msg *MsgUpdateParams) String() string {
	return fmt.Sprintf("MsgUpdateParams{Signer: %s, Params: %s}",
		msg.Signer, msg.Params.String())
}

// ProtoMessage implements proto.Message
func (*MsgUpdateParams) ProtoMessage() {}

// NewMsgUpdateParams creates a new MsgUpdateParams instance
func NewMsgUpdateParams(signer string, params Params) *MsgUpdateParams {
	return &MsgUpdateParams{
		Signer: signer,
		Params: params,
	}
}

// Route implements sdk.Msg
func (msg MsgUpdateParams) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg MsgUpdateParams) Type() string {
	return TypeMsgUpdateParams
}

// GetSigners implements sdk.Msg
func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

// GetSignBytes implements sdk.Msg
func (msg *MsgUpdateParams) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg
func (msg *MsgUpdateParams) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}

	return msg.Params.Validate()
}
