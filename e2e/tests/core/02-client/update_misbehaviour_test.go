//go:build !test_e2e

package client

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	test "github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cometbft/cometbft/privval"
	cmttypes "github.com/cometbft/cometbft/types"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/ibc-go/e2e/dockerutil"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
)

func TestParamsClientTestSuite(t *testing.T) {
	testifysuite.Run(t, new(ParamsClientTestSuite))
}

type ParamsClientTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *ParamsClientTestSuite) SetupSuite() {
	chainA, chainB := s.GetChains()
	s.SetChainsIntoSuite(chainA, chainB)
}
func (s *ParamsClientTestSuite) TestClient_Update_Misbehaviour() {
	t := s.T()
	t.Parallel()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	relayer, _ := s.SetupRelayer(ctx, s.TransferChannelOptions(), chainA, chainB)

	var (
		trustedHeight   clienttypes.Height
		latestHeight    clienttypes.Height
		clientState     ibcexported.ClientState
		header          testsuite.Header
		signers         []cmttypes.PrivValidator
		validatorSet    []*cmttypes.Validator
		maliciousHeader *ibctm.Header
		err             error
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 10, chainA, chainB))

	t.Run("update clients", func(t *testing.T) {
		err := relayer.UpdateClients(ctx, s.GetRelayerExecReporter(), s.GetPathNameFromSuite(relayer))
		s.Require().NoError(err)

		clientState, err = s.QueryClientState(ctx, chainA, ibctesting.FirstClientID)
		s.Require().NoError(err)
	})

	t.Run("fetch trusted height", func(t *testing.T) {
		tmClientState, ok := clientState.(*ibctm.ClientState)
		s.Require().True(ok)

		trustedHeight, ok = tmClientState.GetLatestHeight().(clienttypes.Height)
		s.Require().True(ok)
	})

	t.Run("update clients", func(t *testing.T) {
		err := relayer.UpdateClients(ctx, s.GetRelayerExecReporter(), s.GetPathNameFromSuite(relayer))
		s.Require().NoError(err)

		clientState, err = s.QueryClientState(ctx, chainA, ibctesting.FirstClientID)
		s.Require().NoError(err)
	})

	t.Run("fetch client state latest height", func(t *testing.T) {
		tmClientState, ok := clientState.(*ibctm.ClientState)
		s.Require().True(ok)

		latestHeight, ok = tmClientState.GetLatestHeight().(clienttypes.Height)
		s.Require().True(ok)
	})

	t.Run("create validator set", func(t *testing.T) {
		var validators []*cmtservice.Validator

		t.Run("fetch block header at latest client state height", func(t *testing.T) {
			header, err = s.GetBlockHeaderByHeight(ctx, chainB, latestHeight.GetRevisionHeight())
			s.Require().NoError(err)
		})

		t.Run("get validators at latest height", func(t *testing.T) {
			validators, err = s.GetValidatorSetByHeight(ctx, chainB, latestHeight.GetRevisionHeight())
			s.Require().NoError(err)
		})

		t.Run("extract validator private keys", func(t *testing.T) {
			privateKeys := s.extractChainPrivateKeys(ctx, chainB)
			for i, pv := range privateKeys {
				pubKey, err := pv.GetPubKey()
				s.Require().NoError(err)

				validator := cmttypes.NewValidator(pubKey, validators[i].VotingPower)

				validatorSet = append(validatorSet, validator)
				signers = append(signers, pv)
			}
		})
	})

	t.Run("create malicious header", func(t *testing.T) {
		valSet := cmttypes.NewValidatorSet(validatorSet)
		maliciousHeader, err = createMaliciousTMHeader(chainB.Config().ChainID, int64(latestHeight.GetRevisionHeight()), trustedHeight,
			header.GetTime(), valSet, valSet, signers, header)
		s.Require().NoError(err)
	})

	t.Run("update client with duplicate misbehaviour header", func(t *testing.T) {
		rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
		msgUpdateClient, err := clienttypes.NewMsgUpdateClient(ibctesting.FirstClientID, maliciousHeader, rlyWallet.FormattedAddress())
		s.Require().NoError(err)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgUpdateClient)
		s.AssertTxSuccess(txResp)
	})

	t.Run("ensure client status is frozen", func(t *testing.T) {
		status, err := s.QueryClientStatus(ctx, chainA, ibctesting.FirstClientID)
		s.Require().NoError(err)
		s.Require().Equal(ibcexported.Frozen.String(), status)
	})
}

// extractChainPrivateKeys returns a slice of cmttypes.PrivValidator which hold the private keys for all validator
// nodes for a given chain.
func (s *ParamsClientTestSuite) extractChainPrivateKeys(ctx context.Context, chain ibc.Chain) []cmttypes.PrivValidator {
	testContainers, err := dockerutil.GetTestContainers(ctx, s.T(), s.DockerClient)
	s.Require().NoError(err)

	var filePvs []privval.FilePVKey
	var pvs []cmttypes.PrivValidator
	for _, container := range testContainers {
		isNodeForDifferentChain := !strings.Contains(container.Names[0], chain.Config().ChainID)
		isFullNode := strings.Contains(container.Names[0], fmt.Sprintf("%s-fn", chain.Config().ChainID))
		if isNodeForDifferentChain || isFullNode {
			continue
		}

		validatorPrivKey := fmt.Sprintf("/var/cosmos-chain/%s/config/priv_validator_key.json", chain.Config().Name)
		privKeyFileContents, err := dockerutil.GetFileContentsFromContainer(ctx, s.DockerClient, container.ID, validatorPrivKey)
		s.Require().NoError(err)

		var filePV privval.FilePVKey
		err = cmtjson.Unmarshal(privKeyFileContents, &filePV)
		s.Require().NoError(err)
		filePvs = append(filePvs, filePV)
	}

	// We sort by address as GetValidatorSetByHeight also sorts by address. When iterating over them, the index
	// will correspond to the correct ibcmock.PV.
	sort.SliceStable(filePvs, func(i, j int) bool {
		return filePvs[i].Address.String() < filePvs[j].Address.String()
	})

	for _, filePV := range filePvs {
		pvs = append(pvs, &ibcmock.PV{
			PrivKey: &ed25519.PrivKey{Key: filePV.PrivKey.Bytes()},
		})
	}

	return pvs
}
