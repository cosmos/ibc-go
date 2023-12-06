//go:build !test_e2e

package client

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
)

func TestRecoverClientTestSuite(t *testing.T) {
	testifysuite.Run(t, new(RecoverClientTestSuite))
}

type RecoverClientTestSuite struct {
	testsuite.E2ETestSuite
	chainA ibc.Chain
	chainB ibc.Chain
	rly    ibc.Relayer
}

func (s *RecoverClientTestSuite) SetupTest() {
	ctx := context.TODO()
	s.chainA, s.chainB = s.GetChains()
	s.rly = s.SetupRelayer(ctx, s.TransferChannelOptions(), s.chainA, s.chainB)
}

// Status queries the current status of the client
func (s *RecoverClientTestSuite) Status(ctx context.Context, chain ibc.Chain, clientID string) (string, error) {
	queryClient := s.GetChainGRCPClients(chain).ClientQueryClient
	res, err := queryClient.ClientStatus(ctx, &clienttypes.QueryClientStatusRequest{
		ClientId: clientID,
	})
	if err != nil {
		return "", err
	}

	return res.Status, nil
}

// TestRecoverClient_Succeeds tests that a governance proposal to recover a client using a MsgRecoverClient is successful.
func (s *RecoverClientTestSuite) TestRecoverClient_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	var (
		pathName           string
		subjectClientID    string
		substituteClientID string
		// set the trusting period to a value which will still be valid upon client creation, but invalid before the first update
		badTrustingPeriod = time.Second * 10
	)

	t.Run("create substitute client with correct trusting period", func(t *testing.T) {
		_, err := s.rly.GetChannels(ctx, s.GetRelayerExecReporter(), s.chainA.Config().ChainID)
		s.Require().NoError(err)

		// TODO: update when client identifier created is accessible
		// currently assumes first client is 07-tendermint-0
		substituteClientID = clienttypes.FormatClientIdentifier(ibcexported.Tendermint, 0)

		// TODO: replace with better handling of path names
		pathName = fmt.Sprintf("path-%d", 0)
		pathName = strings.ReplaceAll(pathName, "/", "-")
	})

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount, s.chainA)

	t.Run("create subject client with bad trusting period", func(t *testing.T) {
		createClientOptions := ibc.CreateClientOptions{
			TrustingPeriod: badTrustingPeriod.String(),
		}

		s.SetupClients(ctx, s.rly, createClientOptions)

		// TODO: update when client identifier created is accessible
		// currently assumes second client is 07-tendermint-1
		subjectClientID = clienttypes.FormatClientIdentifier(ibcexported.Tendermint, 1)
	})

	time.Sleep(badTrustingPeriod)

	t.Run("update substitute client", func(t *testing.T) {
		s.UpdateClients(ctx, s.rly, pathName)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 1, s.chainA, s.chainB), "failed to wait for blocks")

	t.Run("check status of each client", func(t *testing.T) {
		t.Run("substitute should be active", func(t *testing.T) {
			status, err := s.Status(ctx, s.chainA, substituteClientID)
			s.Require().NoError(err)
			s.Require().Equal(ibcexported.Active.String(), status)
		})

		t.Run("subject should be expired", func(t *testing.T) {
			status, err := s.Status(ctx, s.chainA, subjectClientID)
			s.Require().NoError(err)
			s.Require().Equal(ibcexported.Expired.String(), status)
		})
	})

	t.Run("execute proposal for MsgRecoverClient", func(t *testing.T) {
		authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, s.chainA)
		s.Require().NoError(err)
		recoverClientMsg := clienttypes.NewMsgRecoverClient(authority.String(), subjectClientID, substituteClientID)
		s.Require().NotNil(recoverClientMsg)
		s.ExecuteAndPassGovV1Proposal(ctx, recoverClientMsg, s.chainA, chainAWallet)
	})

	t.Run("check status of each client", func(t *testing.T) {
		t.Run("substitute should be active", func(t *testing.T) {
			status, err := s.Status(ctx, s.chainA, substituteClientID)
			s.Require().NoError(err)
			s.Require().Equal(ibcexported.Active.String(), status)
		})

		t.Run("subject should be active", func(t *testing.T) {
			status, err := s.Status(ctx, s.chainA, subjectClientID)
			s.Require().NoError(err)
			s.Require().Equal(ibcexported.Active.String(), status)
		})
	})
}
