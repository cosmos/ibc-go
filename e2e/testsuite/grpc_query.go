package testsuite

import (
	"context"

	intertxtypes "github.com/cosmos/interchain-accounts/x/inter-tx/types"
	"github.com/strangelove-ventures/ibctest/ibc"

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
