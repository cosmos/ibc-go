package solomachine_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	clienttypes "github.com/cosmos/ibc-go/v6/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v6/modules/core/03-connection/types"
	commitmenttypes "github.com/cosmos/ibc-go/v6/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v6/modules/core/24-host"
	"github.com/cosmos/ibc-go/v6/modules/core/exported"
	solomachine "github.com/cosmos/ibc-go/v6/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v6/testing"
)

type SoloMachineTestSuite struct {
	suite.Suite

	solomachine      *ibctesting.Solomachine // singlesig public key
	solomachineMulti *ibctesting.Solomachine // multisig public key
	coordinator      *ibctesting.Coordinator

	// testing chain used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	store sdk.KVStore
}

func (suite *SoloMachineTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	suite.solomachine = ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachinesingle", "testing", 1)
	suite.solomachineMulti = ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachinemulti", "testing", 4)

	suite.store = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), exported.Solomachine)
}

func TestSoloMachineTestSuite(t *testing.T) {
	suite.Run(t, new(SoloMachineTestSuite))
}

func (suite *SoloMachineTestSuite) TestConnectionHandshake() {
	var (
		counterpartyClientID     = ibctesting.FirstClientID
		counterpartyConnectionID = ibctesting.FirstConnectionID
	)

	// create solomachine on-chain client
	msgCreateClient, err := clienttypes.NewMsgCreateClient(suite.solomachine.ClientState(), suite.solomachine.ConsensusState(), suite.chainA.SenderAccount.GetAddress().String())
	suite.Require().NoError(err)

	res, err := suite.chainA.SendMsgs(msgCreateClient)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	clientID, err := ibctesting.ParseClientIDFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	// open init
	msgConnOpenInit := connectiontypes.NewMsgConnectionOpenInit(
		clientID,
		counterpartyClientID,
		suite.chainA.GetPrefix(), ibctesting.DefaultOpenInitVersion, ibctesting.DefaultDelayPeriod,
		suite.chainA.SenderAccount.GetAddress().String(),
	)

	res, err = suite.chainA.SendMsgs(msgConnOpenInit)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	connectionID, err := ibctesting.ParseConnectionIDFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	// open try is not necessary as the solo machine implementation is mock'd

	// open ack
	proofTry := suite.solomachine.GenerateConnOpenTryProof(clientID, connectionID)

	clientState := ibctm.NewClientState(suite.chainA.ChainID, ibctesting.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, suite.chainA.LastHeader.GetHeight().(clienttypes.Height), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath)
	proofClient := suite.solomachine.GenerateClientStateProof(clientState)

	consensusState := suite.chainA.LastHeader.ConsensusState()
	consensusHeight := suite.chainA.LastHeader.GetHeight()
	proofConsensus := suite.solomachine.GenerateConsensusStateProof(consensusState, consensusHeight)

	msgConnOpenAck := connectiontypes.NewMsgConnectionOpenAck(
		connectionID, counterpartyConnectionID, clientState, // testing doesn't use flexible selection
		proofTry, proofClient, proofConsensus,
		clienttypes.ZeroHeight(), clientState.GetLatestHeight().(clienttypes.Height),
		ibctesting.ConnectionVersion,
		suite.chainA.SenderAccount.GetAddress().String(),
	)
	res, err = suite.chainA.SendMsgs(msgConnOpenAck)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// open ack is not necessary as the solo machine implementation is mock'd
}

func (suite *SoloMachineTestSuite) GetSequenceFromStore() uint64 {
	bz := suite.store.Get(host.ClientStateKey())
	suite.Require().NotNil(bz)

	var clientState exported.ClientState
	err := suite.chainA.Codec.UnmarshalInterface(bz, &clientState)
	suite.Require().NoError(err)
	return clientState.GetLatestHeight().GetRevisionHeight()
}

func (suite *SoloMachineTestSuite) GetInvalidProof() []byte {
	invalidProof, err := suite.chainA.Codec.Marshal(&solomachine.TimestampedSignatureData{Timestamp: suite.solomachine.Time})
	suite.Require().NoError(err)

	return invalidProof
}

func TestUnpackInterfaces_Header(t *testing.T) {
	registry := testdata.NewTestInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)

	pk := secp256k1.GenPrivKey().PubKey()
	any, err := codectypes.NewAnyWithValue(pk)
	require.NoError(t, err)

	header := solomachine.Header{
		NewPublicKey: any,
	}
	bz, err := header.Marshal()
	require.NoError(t, err)

	var header2 solomachine.Header
	err = header2.Unmarshal(bz)
	require.NoError(t, err)

	err = codectypes.UnpackInterfaces(header2, registry)
	require.NoError(t, err)

	require.Equal(t, pk, header2.NewPublicKey.GetCachedValue())
}

func TestUnpackInterfaces_HeaderData(t *testing.T) {
	registry := testdata.NewTestInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)

	pk := secp256k1.GenPrivKey().PubKey()
	any, err := codectypes.NewAnyWithValue(pk)
	require.NoError(t, err)

	hd := solomachine.HeaderData{
		NewPubKey: any,
	}
	bz, err := hd.Marshal()
	require.NoError(t, err)

	var hd2 solomachine.HeaderData
	err = hd2.Unmarshal(bz)
	require.NoError(t, err)

	err = codectypes.UnpackInterfaces(hd2, registry)
	require.NoError(t, err)

	require.Equal(t, pk, hd2.NewPubKey.GetCachedValue())
}
