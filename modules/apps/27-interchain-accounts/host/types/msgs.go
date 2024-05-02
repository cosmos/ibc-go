package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
)

var _ sdk.Msg = (*MsgModuleQuerySafe)(nil)

// NewMsgModuleQuerySafe creates a new MsgModuleQuerySafe instance
func NewMsgModuleQuerySafe(signer string, requests []*QueryRequest) *MsgModuleQuerySafe {
	return &MsgModuleQuerySafe{
		Signer:   signer,
		Requests: requests,
	}
}

// ValidateBasic implements sdk.HasValidateBasic
func (msg MsgModuleQuerySafe) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	if len(msg.Requests) == 0 {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "no queries provided")
	}

	return nil
}

// GetSigners implements sdk.Msg
func (msg MsgModuleQuerySafe) GetSigners() []sdk.AccAddress {
	accAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{accAddr}
}
