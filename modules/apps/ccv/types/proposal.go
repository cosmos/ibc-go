package types

import (
	"fmt"
	"strings"
	time "time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/modules/core/exported"
)

const (
	ProposalTypeCreateChildChain = "CreateChildChain"
)

var (
	_ govtypes.Content = &CreateChildChainProposal{}
)

func init() {
	govtypes.RegisterProposalType(ProposalTypeCreateChildChain)
}

// NewCreateChildChainProposal creates a new create childchain proposal.
func NewCreateChildChainProposal(title, description, chainID string, clientState exported.ClientState, genesisHash []byte, spawnTime time.Time) (govtypes.Content, error) {
	any, err := clienttypes.PackClientState(clientState)
	if err != nil {
		return nil, err
	}
	return &CreateChildChainProposal{
		Title:       title,
		Description: description,
		ChainId:     chainID,
		ClientState: any,
		GenesisHash: genesisHash,
		SpawnTime:   spawnTime,
	}, nil
}

// GetTitle returns the title of a create childchain proposal.
func (cccp *CreateChildChainProposal) GetTitle() string { return cccp.Title }

// GetDescription returns the description of a create childchain proposal.
func (cccp *CreateChildChainProposal) GetDescription() string { return cccp.Description }

// ProposalRoute returns the routing key of a create childchain proposal.
func (cccp *CreateChildChainProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type of a create childchain proposal.
func (cccp *CreateChildChainProposal) ProposalType() string { return ProposalTypeCreateChildChain }

// ValidateBasic runs basic stateless validity checks
func (cccp *CreateChildChainProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(cccp); err != nil {
		return err
	}

	if strings.TrimSpace(cccp.ChainId) == "" {
		return sdkerrors.Wrap(ErrInvalidProposal, "child chain id must not be blank")
	}

	if cccp.ClientState == nil {
		return sdkerrors.Wrap(ErrInvalidProposal, "child client state cannot be nil")
	}

	_, err := clienttypes.UnpackClientState(cccp.ClientState)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to unpack child client state")
	}

	if len(cccp.GenesisHash) == 0 {
		return sdkerrors.Wrap(ErrInvalidProposal, "genesis hash cannot be empty")
	}

	if cccp.SpawnTime.IsZero() {
		return sdkerrors.Wrap(ErrInvalidProposal, "spawn time cannot be zero")
	}
	return nil
}

// String returns the string representation of the CreateChildChainProposal.
func (cccp *CreateChildChainProposal) String() string {
	var childClientStr string
	upgradedClient, err := clienttypes.UnpackClientState(cccp.ClientState)
	if err != nil {
		childClientStr = "invalid IBC Client State"
	} else {
		childClientStr = upgradedClient.String()
	}

	return fmt.Sprintf(`CreateChildChain Proposal
	Title: %s
	Description: %s
	ChainID: %s
	ClientState: %s
	GenesisHash: %s
	SpawnTime: %s`, cccp.Title, cccp.Description, cccp.ChainId, childClientStr, cccp.GenesisHash, cccp.SpawnTime)
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (cccp CreateChildChainProposal) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return unpacker.UnpackAny(cccp.ClientState, new(exported.ClientState))
}
