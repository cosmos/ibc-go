//go:build !test_e2e

package client

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cometbft/cometbft/crypto/tmhash"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtprotoversion "github.com/cometbft/cometbft/proto/tendermint/version"
	cmttypes "github.com/cometbft/cometbft/types"
	cmtversion "github.com/cometbft/cometbft/version"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

const (
	invalidHashValue = "invalid_hash"
)

func TestClientTestSuite(t *testing.T) {
	testifysuite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *ClientTestSuite) SetupSuite() {
	chainA, chainB := s.GetChains()
	s.SetChainsIntoSuite(chainA, chainB)
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

// TestScheduleIBCUpgrade_Succeeds tests that a governance proposal to schedule an IBC software upgrade is successful.
func (s *ClientTestSuite) TestScheduleIBCUpgrade_Succeeds() {
	t := s.T()
	t.Parallel()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	const planHeight = int64(300)
	const legacyPlanHeight = planHeight * 2
	var newChainID string

	t.Run("setup relayer", func(t *testing.T) {
		_, _ = s.SetupRelayer(ctx, s.TransferChannelOptions(), chainA, chainB)
	})

	t.Run("execute proposal for MsgIBCSoftwareUpgrade", func(t *testing.T) {
		authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
		s.Require().NoError(err)
		s.Require().NotNil(authority)

		clientState, err := s.QueryClientState(ctx, chainB, ibctesting.FirstClientID)
		s.Require().NoError(err)

		originalChainID := clientState.(*ibctm.ClientState).ChainId
		revisionNumber := clienttypes.ParseChainID(originalChainID)
		// increment revision number even with new chain ID to prevent loss of misbehaviour detection support
		newChainID, err = clienttypes.SetRevisionNumber(originalChainID, revisionNumber+1)
		s.Require().NoError(err)
		s.Require().NotEqual(originalChainID, newChainID)

		upgradedClientState := clientState.(*ibctm.ClientState)
		upgradedClientState.ChainId = newChainID

		scheduleUpgradeMsg, err := clienttypes.NewMsgIBCSoftwareUpgrade(
			authority.String(),
			upgradetypes.Plan{
				Name:   "upgrade-client",
				Height: planHeight,
			},
			upgradedClientState,
		)
		s.Require().NoError(err)
		s.ExecuteAndPassGovV1Proposal(ctx, scheduleUpgradeMsg, chainA, chainAWallet)
	})

	t.Run("check that IBC software upgrade has been scheduled successfully on chainA", func(t *testing.T) {
		// checks there is an upgraded client state stored
		cs, err := s.QueryUpgradedClientState(ctx, chainA, ibctesting.FirstClientID)
		s.Require().NoError(err)

		upgradedClientState, ok := cs.(*ibctm.ClientState)
		s.Require().True(ok)
		s.Require().Equal(upgradedClientState.ChainId, newChainID)

		plan, err := s.QueryCurrentUpgradePlan(ctx, chainA)
		s.Require().NoError(err)

		s.Require().Equal("upgrade-client", plan.Name)
		s.Require().Equal(planHeight, plan.Height)
	})

	t.Run("ensure legacy proposal does not succeed", func(t *testing.T) {
		authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
		s.Require().NoError(err)
		s.Require().NotNil(authority)

		clientState, err := s.QueryClientState(ctx, chainB, ibctesting.FirstClientID)
		s.Require().NoError(err)

		originalChainID := clientState.(*ibctm.ClientState).ChainId
		revisionNumber := clienttypes.ParseChainID(originalChainID)
		// increment revision number even with new chain ID to prevent loss of misbehaviour detection support
		newChainID, err = clienttypes.SetRevisionNumber(originalChainID, revisionNumber+1)
		s.Require().NoError(err)
		s.Require().NotEqual(originalChainID, newChainID)

		upgradedClientState := clientState.ZeroCustomFields().(*ibctm.ClientState)
		upgradedClientState.ChainId = newChainID

		legacyUpgradeProposal, err := clienttypes.NewUpgradeProposal(ibctesting.Title, ibctesting.Description, upgradetypes.Plan{
			Name:   "upgrade-client-legacy",
			Height: legacyPlanHeight,
		}, upgradedClientState)

		s.Require().NoError(err)
		txResp := s.ExecuteGovV1Beta1Proposal(ctx, chainA, chainAWallet, legacyUpgradeProposal)
		s.AssertTxFailure(txResp, govtypes.ErrInvalidProposalContent)
	})
}

func (s *ClientTestSuite) TestClientUpdateProposal_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()

	var (
		pathName           string
		subjectClientID    string
		substituteClientID string
		// set the trusting period to a value which will still be valid upon client creation, but invalid before the first update
		badTrustingPeriod = time.Second * 10
	)

	t.Run("setup relayer", func(t *testing.T) {
		relayer, _ := s.SetupRelayer(ctx, s.TransferChannelOptions(), chainA, chainB)
		// TODO: replace with better handling of path names
		pathName = s.GetPathNameFromSuite(relayer)
		substituteClientID = strings.ReplaceAll(pathName, "path", ibcexported.Tendermint)
	})

	t.Run("create subject client with bad trusting period", func(t *testing.T) {
		createClientOptions := ibc.CreateClientOptions{
			TrustingPeriod: badTrustingPeriod.String(),
		}

		s.SetupClients(ctx, s.GetRelayerFromPath(pathName), createClientOptions)
		subjectClientID = clienttypes.FormatClientIdentifier(ibcexported.Tendermint, uint64(s.GetPathNameIndexFromSuite()-1))

		time.Sleep(badTrustingPeriod)
	})

	t.Run("update substitute client", func(t *testing.T) {
		s.UpdateClients(ctx, s.GetRelayerFromPath(pathName), pathName)
	})

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
		chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

		proposal := clienttypes.NewClientUpdateProposal(ibctesting.Title, ibctesting.Description, subjectClientID, substituteClientID)
		s.ExecuteAndPassGovV1Beta1Proposal(ctx, chainA, chainAWallet, proposal)
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

// TestRecoverClient_Succeeds tests that a governance proposal to recover a client using a MsgRecoverClient is successful.
func (s *ClientTestSuite) TestRecoverClient_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()

	var (
		pathName           string
		subjectClientID    string
		substituteClientID string
		// set the trusting period to a value which will still be valid upon client creation, but invalid before the first update
		badTrustingPeriod = time.Second * 10
	)

	t.Run("setup relayer", func(t *testing.T) {
		relayer, _ := s.SetupRelayer(ctx, s.TransferChannelOptions(), chainA, chainB)
		// TODO: replace with better handling of path names
		pathName = s.GetPathNameFromSuite(relayer)
		substituteClientID = strings.ReplaceAll(pathName, "path", ibcexported.Tendermint)
	})

	t.Run("create subject client with bad trusting period", func(t *testing.T) {
		createClientOptions := ibc.CreateClientOptions{
			TrustingPeriod: badTrustingPeriod.String(),
		}

		s.SetupClients(ctx, s.GetRelayerFromPath(pathName), createClientOptions)
		subjectClientID = clienttypes.FormatClientIdentifier(ibcexported.Tendermint, uint64(s.GetPathNameIndexFromSuite()-1))
		time.Sleep(badTrustingPeriod)
	})

	t.Run("update substitute client", func(t *testing.T) {
		s.UpdateClients(ctx, s.GetRelayerFromPath(pathName), pathName)
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

	t.Run("execute proposal for MsgRecoverClient", func(t *testing.T) {
		s.Require().NoError(test.WaitForBlocks(ctx, 15, chainA, chainB), "failed to wait for blocks")
		chainAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
		authority, err := s.QueryModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
		s.Require().NoError(err)
		recoverClientMsg := clienttypes.NewMsgRecoverClient(authority.String(), subjectClientID, substituteClientID)
		s.Require().NotNil(recoverClientMsg)
		s.ExecuteAndPassGovV1Proposal(ctx, recoverClientMsg, chainA, chainAWallet)
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

// createMaliciousTMHeader creates a header with the provided trusted height with an invalid app hash.
func createMaliciousTMHeader(chainID string, blockHeight int64, trustedHeight clienttypes.Height, timestamp time.Time, tmValSet, tmTrustedVals *cmttypes.ValidatorSet, signers []cmttypes.PrivValidator, oldHeader testsuite.Header) (*ibctm.Header, error) {
	tmHeader := cmttypes.Header{
		Version:            cmtprotoversion.Consensus{Block: cmtversion.BlockProtocol, App: 2},
		ChainID:            chainID,
		Height:             blockHeight,
		Time:               timestamp,
		LastBlockID:        ibctesting.MakeBlockID(make([]byte, tmhash.Size), 10_000, make([]byte, tmhash.Size)),
		LastCommitHash:     oldHeader.GetLastCommitHash(),
		ValidatorsHash:     tmValSet.Hash(),
		NextValidatorsHash: tmValSet.Hash(),
		DataHash:           tmhash.Sum([]byte(invalidHashValue)),
		ConsensusHash:      tmhash.Sum([]byte(invalidHashValue)),
		AppHash:            tmhash.Sum([]byte(invalidHashValue)),
		LastResultsHash:    tmhash.Sum([]byte(invalidHashValue)),
		EvidenceHash:       tmhash.Sum([]byte(invalidHashValue)),
		ProposerAddress:    tmValSet.Proposer.Address, //nolint:staticcheck
	}

	hhash := tmHeader.Hash()
	blockID := ibctesting.MakeBlockID(hhash, 3, tmhash.Sum([]byte(invalidHashValue)))
	voteSet := cmttypes.NewVoteSet(chainID, blockHeight, 1, cmtproto.PrecommitType, tmValSet)

	extCommit, err := cmttypes.MakeExtCommit(blockID, blockHeight, 1, voteSet, signers, timestamp, false)
	if err != nil {
		return nil, err
	}

	signedHeader := &cmtproto.SignedHeader{
		Header: tmHeader.ToProto(),
		Commit: extCommit.ToCommit().ToProto(),
	}

	valSet, err := tmValSet.ToProto()
	if err != nil {
		return nil, err
	}

	trustedVals, err := tmTrustedVals.ToProto()
	if err != nil {
		return nil, err
	}

	return &ibctm.Header{
		SignedHeader:      signedHeader,
		ValidatorSet:      valSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: trustedVals,
	}, nil
}
