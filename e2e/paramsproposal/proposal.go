// SPDX-License-Identifier: Apache-2.0

package paramsproposal

import (
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const ProposalTypeChange = "ParameterChange"

var _ govtypes.Content = &ParameterChangeProposal{}

func init() {
	govtypes.RegisterProposalType(ProposalTypeChange)
}

func NewParameterChangeProposal(title, description string, changes []ParamChange) *ParameterChangeProposal {
	return &ParameterChangeProposal{Title: title, Description: description, Changes: changes}
}

func (pcp *ParameterChangeProposal) GetTitle() string       { return pcp.Title }
func (pcp *ParameterChangeProposal) GetDescription() string { return pcp.Description }
func (*ParameterChangeProposal) ProposalRoute() string      { return RouterKey }
func (*ParameterChangeProposal) ProposalType() string       { return ProposalTypeChange }

func (pcp *ParameterChangeProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(pcp); err != nil {
		return err
	}
	return ValidateChanges(pcp.Changes)
}

func NewParamChange(subspace, key, value string) ParamChange {
	return ParamChange{Subspace: subspace, Key: key, Value: value}
}

func ValidateChanges(changes []ParamChange) error {
	if len(changes) == 0 {
		return ErrEmptyChanges
	}
	for _, pc := range changes {
		if len(pc.Subspace) == 0 {
			return ErrEmptySubspace
		}
		if len(pc.Key) == 0 {
			return ErrEmptyKey
		}
		if len(pc.Value) == 0 {
			return ErrEmptyValue
		}
	}
	return nil
}
