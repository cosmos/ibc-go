package testsuite

import (
	"context"
	"sort"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypesbeta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"

	controllertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	feetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
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

	cfg := EncodingConfig()
	var clientState ibcexported.ClientState
	if err := cfg.InterfaceRegistry.UnpackAny(res.ClientState, &clientState); err != nil {
		return nil, err
	}

	return clientState, nil
}

// QueryClientStatus queries the status of the client by clientID
func (s *E2ETestSuite) QueryClientStatus(ctx context.Context, chain ibc.Chain, clientID string) (string, error) {
	queryClient := s.GetChainGRCPClients(chain).ClientQueryClient
	res, err := queryClient.ClientStatus(ctx, &clienttypes.QueryClientStatusRequest{
		ClientId: clientID,
	})
	if err != nil {
		return "", err
	}

	return res.Status, nil
}

// QueryConnection queries the connection end using the given chain and connection id.
func (s *E2ETestSuite) QueryConnection(ctx context.Context, chain ibc.Chain, connectionID string) (connectiontypes.ConnectionEnd, error) {
	queryClient := s.GetChainGRCPClients(chain).ConnectionQueryClient
	res, err := queryClient.Connection(ctx, &connectiontypes.QueryConnectionRequest{
		ConnectionId: connectionID,
	})
	if err != nil {
		return connectiontypes.ConnectionEnd{}, err
	}

	return *res.Connection, nil
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

// GetBlockByHeight fetches the block at a given height. Note: we are explicitly using the res.Block type which has been
// deprecated instead of res.SdkBlock to support backwards compatibility tests.
func (s *E2ETestSuite) GetBlockByHeight(ctx context.Context, chain ibc.Chain, height uint64) (*tmproto.Block, error) {
	tmService := s.GetChainGRCPClients(chain).ConsensusServiceClient
	res, err := tmService.GetBlockByHeight(ctx, &tmservice.GetBlockByHeightRequest{
		Height: int64(height),
	})
	if err != nil {
		return nil, err
	}

	return res.Block, nil
}

// GetValidatorSetByHeight returns the validators of the given chain at the specified height. The returned validators
// are sorted by address.
func (s *E2ETestSuite) GetValidatorSetByHeight(ctx context.Context, chain ibc.Chain, height uint64) ([]*tmservice.Validator, error) {
	tmService := s.GetChainGRCPClients(chain).ConsensusServiceClient
	res, err := tmService.GetValidatorSetByHeight(ctx, &tmservice.GetValidatorSetByHeightRequest{
		Height: int64(height),
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(res.Validators, func(i, j int) bool {
		return res.Validators[i].Address < res.Validators[j].Address
	})

	return res.Validators, nil
}
