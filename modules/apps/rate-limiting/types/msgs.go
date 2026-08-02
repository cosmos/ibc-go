package types

import (
	"regexp"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
)

// channelIDRegex is looser than channeltypes.IsValidChannelID, which also caps the sequence at 20 digits and a uint64.
var channelIDRegex = regexp.MustCompile(`^channel-\d+$`)

func validateChannelOrClientID(channelOrClientID string) error {
	if !channelIDRegex.MatchString(channelOrClientID) && !clienttypes.IsValidClientID(channelOrClientID) {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest,
			"invalid channel or client-id (%s), must be of the format 'channel-{N}' or a valid client-id", channelOrClientID)
	}

	return nil
}

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
	if strings.TrimSpace(msg.Signer) == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "missing sender address")
	}

	if msg.Denom == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid denom (%s)", msg.Denom)
	}

	if err := validateChannelOrClientID(msg.ChannelOrClientId); err != nil {
		return err
	}

	return Quota{
		MaxPercentSend: msg.MaxPercentSend,
		MaxPercentRecv: msg.MaxPercentRecv,
		DurationHours:  msg.DurationHours,
	}.validate()
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
	if strings.TrimSpace(msg.Signer) == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "missing sender address")
	}

	if msg.Denom == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid denom (%s)", msg.Denom)
	}

	if err := validateChannelOrClientID(msg.ChannelOrClientId); err != nil {
		return err
	}

	return Quota{
		MaxPercentSend: msg.MaxPercentSend,
		MaxPercentRecv: msg.MaxPercentRecv,
		DurationHours:  msg.DurationHours,
	}.validate()
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
	if strings.TrimSpace(msg.Signer) == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "missing sender address")
	}

	if msg.Denom == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid denom (%s)", msg.Denom)
	}

	return validateChannelOrClientID(msg.ChannelOrClientId)
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
	if strings.TrimSpace(msg.Signer) == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, "missing sender address")
	}

	if msg.Denom == "" {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "invalid denom (%s)", msg.Denom)
	}

	return validateChannelOrClientID(msg.ChannelOrClientId)
}
