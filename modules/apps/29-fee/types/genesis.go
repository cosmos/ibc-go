package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
)

// NewGenesisState creates a 29-fee GenesisState instance.
func NewGenesisState(identifiedFees []IdentifiedPacketFee, feeEnabledChannels []*FeeEnabledChannel, registeredRelayers []*RegisteredRelayerAddress) *GenesisState {
	return &GenesisState{
		IdentifiedFees:     identifiedFees,
		FeeEnabledChannels: feeEnabledChannels,
		RegisteredRelayers: registeredRelayers,
	}
}

// DefaultGenesisState returns a GenesisState with "transfer" as the default PortID.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		IdentifiedFees:     []IdentifiedPacketFee{},
		FeeEnabledChannels: []*FeeEnabledChannel{},
		RegisteredRelayers: []*RegisteredRelayerAddress{},
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
		_, err := sdk.AccAddressFromBech32(rel.Address)
		if err != nil {
			return sdkerrors.Wrap(err, "failed to convert source relayer address into sdk.AccAddress")
		}

		if rel.CounterpartyAddress == "" {
			return ErrCounterpartyAddressEmpty
		}
	}

	return nil
}
