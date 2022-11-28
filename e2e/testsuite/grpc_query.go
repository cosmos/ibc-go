package testsuite

import (
	"context"

	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypesbeta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"
	"github.com/strangelove-ventures/ibctest/v6/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/v6/ibc"

	controllertypes "github.com/cosmos/ibc-go/v6/modules/apps/27-interchain-accounts/controller/types"
	feetypes "github.com/cosmos/ibc-go/v6/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v6/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v6/modules/core/exported"
)

// QueryClientState queries the client state on the given chain for the provided clientID.
func (s *E2ETestSuite) QueryClientState(ctx context.Context, chain ibc.Chain, clientID string) (ibcexported.ClientState, error) {
	queryClient := s.GetChainGRCPClients(chain).ClientQueryClient
	res, err := queryClient.ClientState(ctx, &clienttypes.QueryClientStateRequest{
		ClientId: clientID,
	})
	if err != nil {
		return nil, err
	}

	clientState, err := clienttypes.UnpackClientState(res.ClientState)
	if err != nil {
		return nil, err
	}

	return clientState, nil
}

// QueryChannel queries the channel on a given chain for the provided portID and channelID
func (s *E2ETestSuite) QueryChannel(ctx context.Context, chain ibc.Chain, portID, channelID string) (channeltypes.Channel, error) {
	queryClient := s.GetChainGRCPClients(chain).ChannelQueryClient
	res, err := queryClient.Channel(ctx, &channeltypes.QueryChannelRequest{
		PortId:    portID,
		ChannelId: channelID,
	})
	if err != nil {
		return channeltypes.Channel{}, err
	}

	return *res.Channel, nil
}

// QueryPacketCommitment queries the packet commitment on the given chain for the provided channel and sequence.
func (s *E2ETestSuite) QueryPacketCommitment(ctx context.Context, chain ibc.Chain, portID, channelID string, sequence uint64) ([]byte, error) {
	queryClient := s.GetChainGRCPClients(chain).ChannelQueryClient
	res, err := queryClient.PacketCommitment(ctx, &channeltypes.QueryPacketCommitmentRequest{
		PortId:    portID,
		ChannelId: channelID,
		Sequence:  sequence,
	})
	if err != nil {
		return nil, err
	}
	return res.Commitment, nil
}

// QueryInterchainAccount queries the interchain account for the given owner and connectionID.
func (s *E2ETestSuite) QueryInterchainAccount(ctx context.Context, chain ibc.Chain, owner, connectionID string) (string, error) {
	queryClient := s.GetChainGRCPClients(chain).ICAQueryClient
	res, err := queryClient.InterchainAccount(ctx, &controllertypes.QueryInterchainAccountRequest{
		Owner:        owner,
		ConnectionId: connectionID,
	})
	if err != nil {
		return "", err
	}
	return res.Address, nil
}

// QueryInterchainAccountLegacy queries the interchain account for the given owner and connectionID using the intertx module.
func (s *E2ETestSuite) QueryInterchainAccountLegacy(ctx context.Context, chain ibc.Chain, owner, connectionID string) (string, error) {
	queryClient := s.GetChainGRCPClients(chain).InterTxQueryClient
	res, err := queryClient.InterchainAccount(ctx, &intertxtypes.QueryInterchainAccountRequest{
		Owner:        owner,
		ConnectionId: connectionID,
	})
	if err != nil {
		return "", err
	}

	return res.InterchainAccountAddress, nil
}

// QueryIncentivizedPacketsForChannel queries the incentivized packets on the specified channel.
func (s *E2ETestSuite) QueryIncentivizedPacketsForChannel(
	ctx context.Context,
	chain *cosmos.CosmosChain,
	portId,
	channelId string,
) ([]*feetypes.IdentifiedPacketFees, error) {
	queryClient := s.GetChainGRCPClients(chain).FeeQueryClient
	res, err := queryClient.IncentivizedPacketsForChannel(ctx, &feetypes.QueryIncentivizedPacketsForChannelRequest{
		PortId:    portId,
		ChannelId: channelId,
	})
	if err != nil {
		return nil, err
	}
	return res.IncentivizedPackets, err
}

// QueryCounterPartyPayee queries the counterparty payee of the given chain and relayer address on the specified channel.
func (s *E2ETestSuite) QueryCounterPartyPayee(ctx context.Context, chain ibc.Chain, relayerAddress, channelID string) (string, error) {
	queryClient := s.GetChainGRCPClients(chain).FeeQueryClient
	res, err := queryClient.CounterpartyPayee(ctx, &feetypes.QueryCounterpartyPayeeRequest{
		ChannelId: channelID,
		Relayer:   relayerAddress,
	})
	if err != nil {
		return "", err
	}
	return res.CounterpartyPayee, nil
}

// QueryProposal queries the governance proposal on the given chain with the given proposal ID.
func (s *E2ETestSuite) QueryProposal(ctx context.Context, chain ibc.Chain, proposalID uint64) (govtypesbeta1.Proposal, error) {
	queryClient := s.GetChainGRCPClients(chain).GovQueryClient
	res, err := queryClient.Proposal(ctx, &govtypesbeta1.QueryProposalRequest{
		ProposalId: proposalID,
	})
	if err != nil {
		return govtypesbeta1.Proposal{}, err
	}

	return res.Proposal, nil
}

func (s *E2ETestSuite) QueryProposalV1(ctx context.Context, chain ibc.Chain, proposalID uint64) (govtypesv1.Proposal, error) {
	queryClient := s.GetChainGRCPClients(chain).GovQueryClientV1
	res, err := queryClient.Proposal(ctx, &govtypesv1.QueryProposalRequest{
		ProposalId: proposalID,
	})
	if err != nil {
		return govtypesv1.Proposal{}, err
	}

	return *res.Proposal, nil
}
