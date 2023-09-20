package types

import (
	errorsmod "cosmossdk.io/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	// ProposalTypeClientUpdate defines the type for a ClientUpdateProposal
	ProposalTypeClientUpdate = "ClientUpdate"
)

var (
	_ govtypes.Content = &ClientUpdateProposal{}
)

func init() {
	govtypes.RegisterProposalType(ProposalTypeClientUpdate)
}

// NewClientUpdateProposal creates a new client update proposal.
func NewClientUpdateProposal(title, description, subjectClientID, substituteClientID string) govtypes.Content {
	return &ClientUpdateProposal{
		Title:              title,
		Description:        description,
		SubjectClientId:    subjectClientID,
		SubstituteClientId: substituteClientID,
	}
}

// GetTitle returns the title of a client update proposal.
func (cup *ClientUpdateProposal) GetTitle() string { return cup.Title }

// GetDescription returns the description of a client update proposal.
func (cup *ClientUpdateProposal) GetDescription() string { return cup.Description }

// ProposalRoute returns the routing key of a client update proposal.
func (cup *ClientUpdateProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type of a client update proposal.
func (cup *ClientUpdateProposal) ProposalType() string { return ProposalTypeClientUpdate }

// ValidateBasic runs basic stateless validity checks
func (cup *ClientUpdateProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(cup)
	if err != nil {
		return err
	}

	if cup.SubjectClientId == cup.SubstituteClientId {
		return errorsmod.Wrap(ErrInvalidSubstitute, "subject and substitute client identifiers are equal")
	}
	if _, _, err := ParseClientIdentifier(cup.SubjectClientId); err != nil {
		return err
	}
	if _, _, err := ParseClientIdentifier(cup.SubstituteClientId); err != nil {
		return err
	}

	return nil
}
