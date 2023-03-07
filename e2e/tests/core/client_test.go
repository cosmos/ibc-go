package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	test "github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/suite"
	"github.com/tendermint/tendermint/crypto/tmhash"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/privval"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmprotoversion "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"
	tmversion "github.com/tendermint/tendermint/version"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/ibc-go/e2e/dockerutil"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
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

func (s *ClientTestSuite) TestClient_Misbehaviour() {
	t := s.T()
	ctx := context.TODO()

	relayer, _ := s.SetupChainsRelayerAndChannel(ctx)
	chainA, chainB := s.GetChains()

	testContainers, err := dockerutil.GetTestContainers(t, ctx, s.DockerClient)
	s.Require().NoError(err)

	t.Logf("found %d containers", len(testContainers))

	var privKeyFileContents []byte
	var misbehavingValidatorContainerIDs []string
	for _, container := range testContainers {
		if !strings.Contains(container.Names[0], chainB.Config().ChainID) { // switched to chain b
			continue
		}
		misbehavingValidatorContainerIDs = append(misbehavingValidatorContainerIDs, container.ID)
		chainBPrivKey := fmt.Sprintf("/var/cosmos-chain/%s/config/priv_validator_key.json", chainB.Config().Name) // switched to chain b

		privKeyFileContents, err = dockerutil.GetFileContentsFromContainer(ctx, s.DockerClient, container.ID, chainBPrivKey)
		s.Require().NoError(err)

		// TODO don't break after first
		break
	}

	var privVal privval.FilePVKey
	err = tmjson.Unmarshal(privKeyFileContents, &privVal)
	s.Require().NoError(err)

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))

	t.Run("update client", func(t *testing.T) {
		// todo: path name is not accessible so manually generating it here
		err := relayer.UpdateClients(ctx, s.GetRelayerExecReporter(), strings.ReplaceAll(fmt.Sprintf("%s-path-%d", s.T().Name(), 0), "/", "-"))
		s.Require().NoError(err)
	})

	t.Run("create misbehaving header", func(t *testing.T) {
		// query client state from chainA and get latest height.
		// then query block by height to get the header info
		clientState, err := s.QueryClientState(ctx, chainA, "07-tendermint-0") // todo: can remove hard coding of client id?
		s.Require().NoError(err)

		tmClientState, ok := clientState.(*ibctm.ClientState)
		s.Require().True(ok)

		// trusted height
		trustedHeight, ok := tmClientState.GetLatestHeight().(clienttypes.Height)
		s.Require().True(ok)

		// -------------- second update client
		err = relayer.UpdateClients(ctx, s.GetRelayerExecReporter(), strings.ReplaceAll(fmt.Sprintf("%s-path-%d", s.T().Name(), 0), "/", "-"))
		s.Require().NoError(err)

		clientState, err = s.QueryClientState(ctx, chainA, "07-tendermint-0") // todo: can remove hard coding of client id?
		s.Require().NoError(err)

		tmClientState, ok = clientState.(*ibctm.ClientState)
		s.Require().True(ok)

		height, ok := tmClientState.GetLatestHeight().(clienttypes.Height)
		s.Require().True(ok)

		tmService := s.GetChainGRCPClients(chainB).ConsensusServiceClient

		blockRes, err := tmService.GetBlockByHeight(ctx, &tmservice.GetBlockByHeightRequest{
			Height: int64(height.GetRevisionHeight()),
		})
		s.Require().NoError(err)

		validatorRes, err := tmService.GetValidatorSetByHeight(ctx, &tmservice.GetValidatorSetByHeightRequest{
			Height: int64(height.GetRevisionHeight()),
		})
		s.Require().NoError(err)

		pv := ibcmock.PV{
			PrivKey: &ed25519.PrivKey{Key: privVal.PrivKey.Bytes()},
		}

		pubKey, err := pv.GetPubKey()
		s.Require().NoError(err)

		validator := tmtypes.NewValidator(pubKey, validatorRes.Validators[0].VotingPower)
		valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})
		signers := []tmtypes.PrivValidator{pv}

		// creating duplicate header
		newHeader := s.createTMClientHeader(t, chainB.Config().ChainID, int64(height.GetRevisionHeight()), trustedHeight,
			blockRes.SdkBlock.Header.GetTime().Add(time.Minute), valSet, valSet, signers, &blockRes.SdkBlock.Header)

		// update client with duplicate header
		rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
		msgUpdateClient, err := clienttypes.NewMsgUpdateClient("07-tendermint-0", newHeader, rlyWallet.FormattedAddress())
		s.Require().NoError(err)

		txResp, err := s.BroadcastMessages(ctx, chainA, rlyWallet, msgUpdateClient)
		s.Require().NoError(err)
		s.AssertValidTxResponse(txResp)

		status, err := s.QueryClientStatus(ctx, chainA, "07-tendermint-0")
		s.Require().NoError(err)
		s.Require().Equal(ibcexported.Frozen.String(), status)
	})
}

// TODO: see what we can clean up here
func (s *ClientTestSuite) createTMClientHeader(t *testing.T, chainID string, blockHeight int64, trustedHeight clienttypes.Height,
	timestamp time.Time, tmValSet, tmTrustedVals *tmtypes.ValidatorSet, signers []tmtypes.PrivValidator,
	oldHeader *tmservice.Header) *ibctm.Header {
	var (
		valSet      *tmproto.ValidatorSet
		trustedVals *tmproto.ValidatorSet
	)
	s.Require().NotNil(t, tmValSet)

	vsetHash := tmValSet.Hash()

	tmHeader := tmtypes.Header{
		Version:            tmprotoversion.Consensus{Block: tmversion.BlockProtocol, App: 2},
		ChainID:            chainID,
		Height:             blockHeight,
		Time:               timestamp,
		LastBlockID:        ibctesting.MakeBlockID(make([]byte, tmhash.Size), 10_000, make([]byte, tmhash.Size)),
		LastCommitHash:     oldHeader.LastCommitHash,
		DataHash:           tmhash.Sum([]byte("data_hash")),
		ValidatorsHash:     vsetHash,
		NextValidatorsHash: vsetHash,
		ConsensusHash:      tmhash.Sum([]byte("consensus_hash")),
		AppHash:            tmhash.Sum([]byte("app_hash")),
		LastResultsHash:    tmhash.Sum([]byte("last_results_hash")),
		EvidenceHash:       tmhash.Sum([]byte("evidence_hash")),
		ProposerAddress:    tmValSet.Proposer.Address, //nolint:staticcheck
	}
	hhash := tmHeader.Hash()
	blockID := ibctesting.MakeBlockID(hhash, 3, tmhash.Sum([]byte("part_set")))
	voteSet := tmtypes.NewVoteSet(chainID, blockHeight, 1, tmproto.PrecommitType, tmValSet)

	commit, err := tmtypes.MakeCommit(blockID, blockHeight, 1, voteSet, signers, timestamp)
	s.Require().NoError(err)

	signedHeader := &tmproto.SignedHeader{
		Header: tmHeader.ToProto(),
		Commit: commit.ToProto(),
	}

	if tmValSet != nil {
		valSet, err = tmValSet.ToProto()
		if err != nil {
			panic(err)
		}
	}

	if tmTrustedVals != nil {
		trustedVals, err = tmTrustedVals.ToProto()
		if err != nil {
			panic(err)
		}
	}

	// The trusted fields may be nil. They may be filled before relaying messages to a client.
	// The relayer is responsible for querying client and injecting appropriate trusted fields.
	return &ibctm.Header{
		SignedHeader:      signedHeader,
		ValidatorSet:      valSet,
		TrustedHeight:     trustedHeight,
		TrustedValidators: trustedVals,
	}
}

// UpdateClients and get trusted height, then update client again.
// Then build duplicate update header, and use first trusted height

// --------------
// cfg := testsuite.EncodingConfig()

// var pk cryptotypes.PubKey
// err = cfg.InterfaceRegistry.UnpackAny(validatorRes.Validators[0].PubKey, &pk)
// s.Require().NoError(err)

// tmPk, err := cryptocodec.ToTmPubKeyInterface(pk)
// s.Require().NoError(err)

// s.Require().Equal(pubKey, tmPk)

// t.Logf("public key address: %s", pubKey.Address())
// t.Logf("public key: %s", pubKey)
// --------------
