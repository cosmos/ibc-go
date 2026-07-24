package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v11/modules/core/24-host"
)

// NewGenesisState creates a new ibc-transfer GenesisState instance.
func NewGenesisState(portID string, denoms Denoms, params Params, totalEscrowed sdk.Coins) *GenesisState {
	return &GenesisState{
		PortId:         portID,
		Denoms:         denoms,
		Params:         params,
		TotalEscrowed:  totalEscrowed,
		ChannelEscrows: []ChannelEscrow{},
	}
}

// DefaultGenesisState returns a GenesisState with "transfer" as the default PortID.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		PortId:         PortID,
		Denoms:         Denoms{},
		Params:         DefaultParams(),
		TotalEscrowed:  sdk.Coins{},
		ChannelEscrows: []ChannelEscrow{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := host.PortIdentifierValidator(gs.PortId); err != nil {
		return err
	}
	if err := gs.Denoms.Validate(); err != nil {
		return err
	}
	if err := gs.TotalEscrowed.Validate(); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(gs.ChannelEscrows))
	var channelTotal sdk.Coins
	for _, escrow := range gs.ChannelEscrows {
		if err := host.ChannelIdentifierValidator(escrow.ChannelOrClientId); err != nil && !clienttypes.IsValidClientID(escrow.ChannelOrClientId) {
			return fmt.Errorf("invalid channel or client identifier %q", escrow.ChannelOrClientId)
		}
		if err := escrow.Tokens.Validate(); err != nil {
			return err
		}

		if _, ok := seen[escrow.ChannelOrClientId]; ok {
			return fmt.Errorf("duplicate channel escrow: %s", escrow.ChannelOrClientId)
		}
		seen[escrow.ChannelOrClientId] = struct{}{}
		channelTotal = channelTotal.Add(escrow.Tokens...)
	}

	if !channelTotal.Equal(gs.TotalEscrowed) {
		return fmt.Errorf("total escrowed %s does not equal channel escrow sum %s", gs.TotalEscrowed, channelTotal)
	}

	return nil
}
