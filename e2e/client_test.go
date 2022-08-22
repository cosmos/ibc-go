package e2e

/*
The ClientTestSuite assumes both chainA and chainB support the ClientUpdate Proposal.
*/

import (
	"context"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v5/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v5/modules/core/exported"
)

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	testsuite.E2ETestSuite
}

// CreateClient broadcasts a MsgCreateClient message.
func (s *TransferTestSuite) CreateClient(ctx context.Context, chain *cosmos.CosmosChain, user *ibctest.User,
	clientState ibcexported.ClientState, counterpartyChain *cosmos.CosmosChain,
) (sdk.TxResponse, error) {
	msg, err := clienttypes.NewMsgCreateClient(clientState, consensusState, user.Bech32Address(chain.Config().Bech32Prefix))
	s.Require().NoError(err)
	return s.BroadcastMessages(ctx, chain, user, msg)
}

func (s *ClientTestSuite) TestClientUpdateProposal_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	t.Run("create substitute client with correct trusting period", func(t *testing.T) {
		relayer, channelA := s.SetupClients(ctx)
		chainA, chainB := s.GetChains()

	})

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	t.Run("create subject client with bad trusting period", func(t *testing.T) {
		clientState = ibctm.NewClientState(
			chainB.ChainID, tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
			height, commitmenttypes.GetSDKSpecs(), UpgradePath)

		createClientTxResp, err := s.CreateClient(ctx, chainA, chainAWallet)
		s.Require().NoError(err)
		s.AssertValidTxResponse(createClientTxResp)
	})

	t.Run("ensure subject client is expired", func(t *testing.T) {
		status := s.Status()
		s.Require().Equal(ibcexported.Expired, status)
	})

	t.Run("pass client update proposal", func(t *testing.T) {
		t.Run("create and submit proposal", func(t *testing.T) {
			prop, err := s.ClientUpdateProposal()
			s.Require().NoError(err)
			s.Require().NotNil(prop)

			err := s.SubmitProposal(prop)
			s.Require().NoError(err)
		})

		t.Run("vote on proposal", func(t *testing.T) {
			err := s.VoteYes(prop)
			s.Require().NoError(err)
		})

		t.Run("wait for proposal to pass", func(t *testing.T) {

		})
	})

	t.Run("ensure subject client has been updated", func(t *testing.T) {

	})
}
