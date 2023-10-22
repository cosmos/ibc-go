package avalanche

import (
	errorsmod "cosmossdk.io/errors"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

var _ exported.ClientMessage = &Header{}

// ConsensusState returns the updated consensus state associated with the header
func (h Header) ConsensusState() *ConsensusState {
	return &ConsensusState{
		Timestamp:          h.SubnetHeader.Timestamp,
		StorageRoot:        h.StorageRoot,
		SignedStorageRoot:  h.SignedStorageRoot,
		ValidatorSet:       h.ValidatorSet,
		SignedValidatorSet: h.SignedValidatorSet,
		Vdrs:               h.Vdrs,
		SignersInput:       h.SignersInput,
	}
}

// ClientType defines that the Header is a Tendermint consensus algorithm
func (h Header) ClientType() string {
	return exported.Avalanche
}

func (h Header) ValidateBasic() error {

	if len(h.SignersInput) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "Avalanche header cannot empty SignersInput")
	}

	if len(h.SignedStorageRoot) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "Avalanche header cannot empty SignedStorageRoot")
	}

	if len(h.StorageRoot) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "Avalanche header cannot empty StorageRoot")
	}

	if len(h.ValidatorSet) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "Avalanche header cannot empty ValidatorSet")
	}

	if len(h.SignedValidatorSet) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "Avalanche header cannot empty SignedValidatorSet")
	}
	if len(h.Vdrs) == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "Avalanche header cannot empty Vdrs")
	}


	if h.SubnetHeader == nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "SubnetHeader is nil")
	}


	if h.PrevSubnetHeader == nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "PrevSubnetHeader is nil")
	}


	if h.PchainHeader == nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "PchainHeader is nil")
	}




	if !h.PrevSubnetHeader.Height.LT(*h.SubnetHeader.Height) {
		return errorsmod.Wrapf(clienttypes.ErrInvalidMisbehaviour, "PrevSubnetHeader height is less or equal than SubnetHeader height (%s <= %s)", h.SubnetHeader.Height, h.SubnetHeader.Height)
	}

	return nil
}
