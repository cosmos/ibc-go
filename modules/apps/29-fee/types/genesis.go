package types

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// NewGenesisState creates a 29-fee GenesisState instance.
func NewGenesisState(identifiedFees []IdentifiedPacketFee, feeEnabledChannels []*FeeEnabledChannel, registeredRelayers []*RegisteredRelayerAddress, forwardRelayers []*ForwardRelayerAddress) *GenesisState {
	return &GenesisState{
		IdentifiedFees:     identifiedFees,
		FeeEnabledChannels: feeEnabledChannels,
		RegisteredRelayers: registeredRelayers,
		ForwardRelayers:    forwardRelayers,
	}
}

// DefaultGenesisState returns a GenesisState with "transfer" as the default PortID.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		IdentifiedFees:     []IdentifiedPacketFee{},
		FeeEnabledChannels: []*FeeEnabledChannel{},
		RegisteredRelayers: []*RegisteredRelayerAddress{},
		ForwardRelayers:    []*ForwardRelayerAddress{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Validate IdentifiedPacketFees
	for _, fee := range gs.IdentifiedFees {
		err := fee.Validate()
		if err != nil {
			return err
		}
	}

	// Validate FeeEnabledChannels
	for _, feeCh := range gs.FeeEnabledChannels {
		if err := host.PortIdentifierValidator(feeCh.PortId); err != nil {
			return sdkerrors.Wrap(err, "invalid source port ID")
		}
		if err := host.ChannelIdentifierValidator(feeCh.ChannelId); err != nil {
			return sdkerrors.Wrap(err, "invalid source channel ID")
		}
	}

	// Validate RegisteredRelayers
	for _, rel := range gs.RegisteredRelayers {
		if _, err := sdk.AccAddressFromBech32(rel.Address); err != nil {
			return sdkerrors.Wrap(err, "failed to convert source relayer address into sdk.AccAddress")
		}

		if strings.TrimSpace(rel.CounterpartyAddress) == "" {
			return ErrCounterpartyAddressEmpty
		}
	}

	// Validate ForwardRelayers
	for _, rel := range gs.ForwardRelayers {
		if _, err := sdk.AccAddressFromBech32(rel.Address); err != nil {
			return sdkerrors.Wrap(err, "failed to convert forward relayer address into sdk.AccAddress")
		}

		if err := rel.PacketId.Validate(); err != nil {
			return err
		}
	}

	return nil
}
