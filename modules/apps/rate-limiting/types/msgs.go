package types

import (
	"regexp"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

var (
	_ sdk.Msg = &MsgAddRateLimit{}
	_ sdk.Msg = &MsgUpdateRateLimit{}
	_ sdk.Msg = &MsgRemoveRateLimit{}
	_ sdk.Msg = &MsgResetRateLimit{}
)

// ----------------------------------------------
//               MsgAddRateLimit
// ----------------------------------------------

func NewMsgAddRateLimit(denom, channelOrClientID string, maxPercentSend sdkmath.Int, maxPercentRecv sdkmath.Int, durationHours uint64) *MsgAddRateLimit {
	return &MsgAddRateLimit{
		Denom:             denom,
		ChannelOrClientId: channelOrClientID,
		MaxPercentSend:    maxPercentSend,
		MaxPercentRecv:    maxPercentRecv,
		DurationHours:     durationHours,
	}
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

func NewMsgUpdateRateLimit(denom, channelOrClientID string, maxPercentSend sdkmath.Int, maxPercentRecv sdkmath.Int, durationHours uint64) *MsgUpdateRateLimit {
	return &MsgUpdateRateLimit{
		Denom:             denom,
		ChannelOrClientId: channelOrClientID,
		MaxPercentSend:    maxPercentSend,
		MaxPercentRecv:    maxPercentRecv,
		DurationHours:     durationHours,
	}
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

func NewMsgRemoveRateLimit(denom, channelOrClientID string) *MsgRemoveRateLimit {
	return &MsgRemoveRateLimit{
		Denom:             denom,
		ChannelOrClientId: channelOrClientID,
	}
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

func NewMsgResetRateLimit(denom, channelOrClientID string) *MsgResetRateLimit {
	return &MsgResetRateLimit{
		Denom:             denom,
		ChannelOrClientId: channelOrClientID,
	}
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
