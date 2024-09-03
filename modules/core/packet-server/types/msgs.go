package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var (
	_ sdk.Msg = (*MsgProvideCounterparty)(nil)

	_ sdk.HasValidateBasic = (*MsgProvideCounterparty)(nil)
)

// NewMsgProvideCounterparty creates a new MsgProvideCounterparty instance
func NewMsgProvideCounterparty(signer, packetPath, clientID string, counterpartyPacketPath commitmenttypes.MerklePath) *MsgProvideCounterparty {
	counterparty := NewCounterparty(clientID, counterpartyPacketPath)

	return &MsgProvideCounterparty{
		Signer:       signer,
		PacketPath:   packetPath,
		Counterparty: counterparty,
	}
}

// ValidateBasic performs basic checks on a MsgProvideCounterparty.
func (msg *MsgProvideCounterparty) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	if strings.TrimSpace(msg.PacketPath) == "" {
		return errorsmod.Wrap(ErrInvalidPacketPath, "packet path cannot be empty")
	}

	if err := msg.Counterparty.Validate(); err != nil {
		return err
	}

	return nil
}
