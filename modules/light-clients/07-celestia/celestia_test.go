package celestia_test

import (
	"encoding/json"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	testifysuite "github.com/stretchr/testify/suite"

	"cosmossdk.io/log"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	ibccelestia "github.com/cosmos/ibc-go/modules/light-clients/07-celestia"
	"github.com/cosmos/ibc-go/modules/light-clients/07-celestia/testing/simapp"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func init() {
	ibctesting.DefaultTestingAppInit = SetupTestingApp
}

// SetupTestingApp provides the duplicated simapp which is specific to the callbacks module on chain creation.
func SetupTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	app := simapp.NewSimApp(log.NewNopLogger(), db, nil, true, simtestutil.EmptyAppOptions{})
	return app, app.DefaultGenesis()
}

type CelestiaTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *CelestiaTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
}

// CreateClient creates an 07-celestia client on a mock cometbft chain.
func (suite *CelestiaTestSuite) CreateClient(endpoint *ibctesting.Endpoint) string {
	tmConfig, ok := endpoint.ClientConfig.(*ibctesting.TendermintConfig)
	suite.Require().True(ok)

	height := endpoint.Counterparty.Chain.LatestCommittedHeader.GetHeight().(clienttypes.Height)
	tmClientState := ibctm.NewClientState(
		endpoint.Counterparty.Chain.ChainID,
		tmConfig.TrustLevel,
		tmConfig.TrustingPeriod,
		tmConfig.UnbondingPeriod,
		tmConfig.MaxClockDrift,
		height,
		commitmenttypes.GetSDKSpecs(),
		ibctesting.UpgradePath,
	)
	clientState := &ibccelestia.ClientState{
		BaseClient: tmClientState,
	}
	tmConsensusState := endpoint.Counterparty.Chain.LatestCommittedHeader.ConsensusState()

	msg, err := clienttypes.NewMsgCreateClient(clientState, tmConsensusState, endpoint.Chain.SenderAccount.GetAddress().String())
	suite.Require().NoError(err)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	clientID, err := ibctesting.ParseClientIDFromEvents(res.Events)
	require.NoError(endpoint.Chain.TB, err)
	endpoint.ClientID = clientID

	return clientID
}

func TestTendermintTestSuite(t *testing.T) {
	testifysuite.Run(t, new(CelestiaTestSuite))
}
