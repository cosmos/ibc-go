package testsuite

import (
	"context"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"

	feetypes "github.com/cosmos/ibc-go/v5/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v5/modules/core/exported"
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

// QueryInterchainAccount queries the interchain account for the given owner and connectionId.
func (s *E2ETestSuite) QueryInterchainAccount(ctx context.Context, chain ibc.Chain, owner, connectionId string) (string, error) {
	queryClient := s.GetChainGRCPClients(chain).ICAQueryClient
	res, err := queryClient.InterchainAccount(ctx, &intertxtypes.QueryInterchainAccountRequest{
		Owner:        owner,
		ConnectionId: connectionId,
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

// QueryProposal
func (s *E2ETestSuite) QueryProposal(ctx context.Context, chain ibc.Chain, proposalID uint64) (govtypes.Proposal, error) {
	queryClient := s.GetChainGRCPClients(chain).GovQueryClient
	res, err := queryClient.Proposal(ctx, &govtypes.QueryProposalRequest{
		ProposalId: proposalID,
	})
	if err != nil {
		return govtypes.Proposal{}, err
	}
	return res.Proposal, nil
}
