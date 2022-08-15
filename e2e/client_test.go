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
)

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *ClientTestSuite) TestClientUpdateProposal_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	relayer, channelA := s.SetupChainsRelayerAndChannel(ctx, transferChannelOptions())
	chainA, chainB := s.GetChains()

	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	chainAAddress := chainAWallet.Bech32Address(chainA.Config().Bech32Prefix)

	t.Run("create subject client with bad trusting period", func(t *testing.T) {

	})

	t.Run("ensure subject client is expired", func(t *testing.T) {

	})

	t.Run("create substitute client with correct trusting period", func(t *testing.T) {

	})

	t.Run("pass client update proposal", func(t *testing.T) {
		t.Run("create and submit proposal", func(t *testing.T) {

		})

		t.Run("vote on proposal", func(t *testing.T) {

		})
		t.Run("wait for proposal to pass", func(t *testing.T) {

		})
	})

	t.Run("ensure subject client has been updated", func(t *testing.T) {

	})
}
