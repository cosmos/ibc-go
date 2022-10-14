package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/strangelove-ventures/ibctest/v6/test"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v6/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	testsuite.E2ETestSuite
}

// Status queries the current status of the client
func (s *ClientTestSuite) Status(ctx context.Context, chain ibc.Chain, clientID string) (string, error) {
	queryClient := s.GetChainGRCPClients(chain).ClientQueryClient
	res, err := queryClient.ClientStatus(ctx, &clienttypes.QueryClientStatusRequest{
		ClientId: clientID,
	})
	if err != nil {
		return "", err
	}

	return res.Status, nil
}

func (s *ClientTestSuite) TestClientUpdateProposal_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	var (
		pathName           string
		relayer            ibc.Relayer
		subjectClientID    string
		substituteClientID string
		badTrustingPeriod  = time.Duration(time.Second)
	)

	t.Run("create substitute client with correct trusting period", func(t *testing.T) {
		relayer, _ = s.SetupChainsRelayerAndChannel(ctx)

		// TODO: update when client identifier created is accessible
		// currently assumes first client is 07-tendermint-0
		substituteClientID = clienttypes.FormatClientIdentifier(ibcexported.Tendermint, 0)

		// TODO: replace with better handling of path names
		pathName = fmt.Sprintf("%s-path-%d", s.T().Name(), 0)
		pathName = strings.ReplaceAll(pathName, "/", "-")
	})

	chainA, chainB := s.GetChains()
	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	t.Run("create subject client with bad trusting period", func(t *testing.T) {
		createClientOptions := ibc.CreateClientOptions{
			TrustingPeriod: badTrustingPeriod.String(),
		}

		s.SetupClients(ctx, relayer, createClientOptions)

		// TODO: update when client identifier created is accessible
		// currently assumes second client is 07-tendermint-1
		subjectClientID = clienttypes.FormatClientIdentifier(ibcexported.Tendermint, 1)
	})

	time.Sleep(badTrustingPeriod)

	t.Run("update substitute client", func(t *testing.T) {
		s.UpdateClients(ctx, relayer, pathName)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	t.Run("check status of each client", func(t *testing.T) {
		t.Run("substitute should be active", func(t *testing.T) {
			status, err := s.Status(ctx, chainA, substituteClientID)
			s.Require().NoError(err)
			s.Require().Equal(ibcexported.Active.String(), status)
		})

		t.Run("subject should be expired", func(t *testing.T) {
			status, err := s.Status(ctx, chainA, subjectClientID)
			s.Require().NoError(err)
			s.Require().Equal(ibcexported.Expired.String(), status)
		})
	})

	t.Run("pass client update proposal", func(t *testing.T) {
		proposal := clienttypes.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subjectClientID, substituteClientID)
		s.ExecuteGovProposal(ctx, chainA, chainAWallet, proposal)
	})

	t.Run("check status of each client", func(t *testing.T) {
		t.Run("substitute should be active", func(t *testing.T) {
			status, err := s.Status(ctx, chainA, substituteClientID)
			s.Require().NoError(err)
			s.Require().Equal(ibcexported.Active.String(), status)
		})

		t.Run("subject should be active", func(t *testing.T) {
			status, err := s.Status(ctx, chainA, subjectClientID)
			s.Require().NoError(err)
			s.Require().Equal(ibcexported.Active.String(), status)
		})
	})
}
