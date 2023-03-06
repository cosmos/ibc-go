package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	test "github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/dockerutil"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	ed25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
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
	_, _ = s.SetupChainsRelayerAndChannel(ctx)
	chainA, _ := s.GetChains()

	testContainers, err := dockerutil.GetTestContainers(t, ctx, s.DockerClient)
	s.Require().NoError(err)

	t.Logf("found %d containers", len(testContainers))

	var privKeyFileContents []byte
	var misbehavingValidatorContainerIDs []string
	for _, container := range testContainers {
		if !strings.Contains(container.Names[0], chainA.Config().ChainID) {
			continue
		}
		misbehavingValidatorContainerIDs = append(misbehavingValidatorContainerIDs, container.ID)
		chainAPrivKey := fmt.Sprintf("/var/cosmos-chain/%s/config/priv_validator_key.json", chainA.Config().Name)

		privKeyFileContents, err = dockerutil.GetFileContentsFromContainer(ctx, s.DockerClient, container.ID, chainAPrivKey)
		s.Require().NoError(err)

		t.Logf("PRIVATE KEY BYTES: %s", string(privKeyFileContents))
		// TODO don't break after first
		break
	}

	type PK struct {
		Value string `json:"value"`
	}

	type PKFile struct {
		PK `json:"priv_key"`
	}


	var pkFile PKFile
	s.Require().NoError(json.Unmarshal(privKeyFileContents, &pkFile))

	pk := &ed25519.PrivKey{}
	s.Require().NoError(pk.UnmarshalAmino([]byte(pkFile.Value)))

	t.Logf("KEY: %s", string(pk.Key))

	//
	//privKey := getSDKPrivKey(1)
	//privVal := ibctestingmock.PV{
	//	PrivKey: privKey,
	//}
	//pubKey, err := privVal.GetPubKey()
	//require.NoError(t, err)
	//validator := tmtypes.NewValidator(pubKey, header.ValidatorSet.Proposer.VotingPower)
	//valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})
	//signers := []tmtypes.PrivValidator{privVal}
	/*
	   {
	     "address": "CA4BCE45283BFB22A4E205305A9D53A6D06E9765",
	     "pub_key": {
	       "type": "tendermint/PubKeyEd25519",
	       "value": "XH0x9cIkK1iyo2o9/lkZ4hTUoze9KpiRgECiSfHYscw="
	     },
	     "priv_key": {
	       "type": "tendermint/PrivKeyEd25519",
	       "value": "c2rFbGDsXY6h1Q0EofeeyOLL32z4HuNMIInrxlEk7JBcfTH1wiQrWLKjaj3+WRniFNSjN70qmJGAQKJJ8dixzA=="
	     }
	   }
	*/



}

func getSDKPrivKey(contents []byte) cryptotypes.PrivKey {
	return ed25519.GenPrivKeyFromSecret(contents)
}
