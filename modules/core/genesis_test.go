package ibc_test

import (
	"errors"
	"fmt"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/codec"

	ibc "github.com/cosmos/ibc-go/v9/modules/core"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v9/modules/core/02-client/v2/types"
	connectiontypes "github.com/cosmos/ibc-go/v9/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channelv2types "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
	"github.com/cosmos/ibc-go/v9/modules/core/types"
	ibctm "github.com/cosmos/ibc-go/v9/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	"github.com/cosmos/ibc-go/v9/testing/simapp"
)

const (
	connectionID  = "connection-0"
	clientID      = "07-tendermint-0"
	connectionID2 = "connection-1"
	clientID2     = "07-tendermint-1"

	port1 = "firstport"
	port2 = "secondport"

	channel1 = "channel-0"
	channel2 = "channel-1"
)

var clientHeight = clienttypes.NewHeight(1, 10)

type IBCTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

// SetupTest creates a coordinator with 2 test chains.
func (suite *IBCTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestIBCTestSuite(t *testing.T) {
	testifysuite.Run(t, new(IBCTestSuite))
}

func (suite *IBCTestSuite) TestValidateGenesis() {
	header := suite.chainA.CreateTMClientHeader(suite.chainA.ChainID, suite.chainA.ProposedHeader.Height, clienttypes.NewHeight(0, uint64(suite.chainA.ProposedHeader.Height-1)), suite.chainA.ProposedHeader.Time, suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Signers)

	testCases := []struct {
		name     string
		genState *types.GenesisState
		expError error
	}{
		{
			name:     "default",
			genState: types.DefaultGenesisState(),
			expError: nil,
		},
		{
			name: "valid genesis",
			genState: &types.GenesisState{
				ClientGenesis: clienttypes.NewGenesisState(
					[]clienttypes.IdentifiedClientState{
						clienttypes.NewIdentifiedClientState(
							clientID, ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
						),
					},
					[]clienttypes.ClientConsensusStates{
						clienttypes.NewClientConsensusStates(
							clientID,
							[]clienttypes.ConsensusStateWithHeight{
								clienttypes.NewConsensusStateWithHeight(
									header.GetHeight().(clienttypes.Height),
									ibctm.NewConsensusState(
										header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.AppHash), header.Header.NextValidatorsHash,
									),
								),
							},
						),
					},
					[]clienttypes.IdentifiedGenesisMetadata{
						clienttypes.NewIdentifiedGenesisMetadata(
							clientID,
							[]clienttypes.GenesisMetadata{
								clienttypes.NewGenesisMetadata([]byte("key1"), []byte("val1")),
								clienttypes.NewGenesisMetadata([]byte("key2"), []byte("val2")),
							},
						),
					},
					clienttypes.NewParams(exported.Tendermint),
					false,
					2,
				),
				ClientV2Genesis: clientv2types.GenesisState{
					CounterpartyInfos: []clientv2types.GenesisCounterpartyInfo{
						{
							ClientId:         "test-1",
							CounterpartyInfo: clientv2types.NewCounterpartyInfo([][]byte{{0o1}}, "test-0"),
						},
						{
							ClientId:         "test-0",
							CounterpartyInfo: clientv2types.NewCounterpartyInfo([][]byte{{0o1}}, "test-1"),
						},
					},
				},
				ConnectionGenesis: connectiontypes.NewGenesisState(
					[]connectiontypes.IdentifiedConnection{
						connectiontypes.NewIdentifiedConnection(connectionID, connectiontypes.NewConnectionEnd(connectiontypes.INIT, clientID, connectiontypes.NewCounterparty(clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))), []*connectiontypes.Version{ibctesting.ConnectionVersion}, 0)),
					},
					[]connectiontypes.ConnectionPaths{
						connectiontypes.NewConnectionPaths(clientID, []string{connectionID}),
					},
					0,
					connectiontypes.NewParams(10),
				),
				ChannelGenesis: channeltypes.NewGenesisState(
					[]channeltypes.IdentifiedChannel{
						channeltypes.NewIdentifiedChannel(
							port1, channel1, channeltypes.NewChannel(
								channeltypes.INIT, channeltypes.ORDERED,
								channeltypes.NewCounterparty(port2, channel2), []string{connectionID}, ibctesting.DefaultChannelVersion,
							),
						),
					},
					[]channeltypes.PacketState{
						channeltypes.NewPacketState(port2, channel2, 1, []byte("ack")),
					},
					[]channeltypes.PacketState{
						channeltypes.NewPacketState(port2, channel2, 1, []byte("")),
					},
					[]channeltypes.PacketState{
						channeltypes.NewPacketState(port1, channel1, 1, []byte("commit_hash")),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port1, channel1, 1),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port2, channel2, 1),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port2, channel2, 1),
					},
					0,
					channeltypes.Params{UpgradeTimeout: channeltypes.DefaultTimeout},
				),
				ChannelV2Genesis: channelv2types.NewGenesisState(
					[]channelv2types.PacketState{
						channelv2types.NewPacketState(channel2, 1, []byte("ack")),
					},
					[]channelv2types.PacketState{
						channelv2types.NewPacketState(channel2, 1, []byte("")),
					},
					[]channelv2types.PacketState{
						channelv2types.NewPacketState(channel1, 1, []byte("commit_hash")),
					},
					[]channelv2types.PacketSequence{
						channelv2types.NewPacketSequence(channel1, 1),
					},
				),
			},
			expError: nil,
		},
		{
			name: "invalid client genesis",
			genState: &types.GenesisState{
				ClientGenesis: clienttypes.NewGenesisState(
					[]clienttypes.IdentifiedClientState{
						clienttypes.NewIdentifiedClientState(
							clientID, ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
						),
					},
					nil,
					[]clienttypes.IdentifiedGenesisMetadata{
						clienttypes.NewIdentifiedGenesisMetadata(
							clientID,
							[]clienttypes.GenesisMetadata{
								clienttypes.NewGenesisMetadata([]byte(""), []byte("val1")),
								clienttypes.NewGenesisMetadata([]byte("key2"), []byte("")),
							},
						),
					},
					clienttypes.NewParams(exported.Tendermint),
					false,
					2,
				),
				ClientV2Genesis:   clientv2types.DefaultGenesisState(),
				ConnectionGenesis: connectiontypes.DefaultGenesisState(),
				ChannelV2Genesis:  channelv2types.DefaultGenesisState(),
			},
			expError: errors.New("genesis metadata key cannot be empty"),
		},
		{
			name: "invalid connection genesis",
			genState: &types.GenesisState{
				ClientGenesis:   clienttypes.DefaultGenesisState(),
				ClientV2Genesis: clientv2types.DefaultGenesisState(),
				ConnectionGenesis: connectiontypes.NewGenesisState(
					[]connectiontypes.IdentifiedConnection{
						connectiontypes.NewIdentifiedConnection(connectionID, connectiontypes.NewConnectionEnd(connectiontypes.INIT, "(CLIENTIDONE)", connectiontypes.NewCounterparty(clientID, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))), []*connectiontypes.Version{connectiontypes.NewVersion("1.1", nil)}, 0)),
					},
					[]connectiontypes.ConnectionPaths{
						connectiontypes.NewConnectionPaths(clientID, []string{connectionID}),
					},
					0,
					connectiontypes.Params{},
				),
				ChannelV2Genesis: channelv2types.DefaultGenesisState(),
			},
			expError: errors.New("invalid connection"),
		},
		{
			name: "invalid channel genesis",
			genState: &types.GenesisState{
				ClientGenesis:     clienttypes.DefaultGenesisState(),
				ClientV2Genesis:   clientv2types.DefaultGenesisState(),
				ConnectionGenesis: connectiontypes.DefaultGenesisState(),
				ChannelGenesis: channeltypes.GenesisState{
					Acknowledgements: []channeltypes.PacketState{
						channeltypes.NewPacketState("(portID)", channel1, 1, []byte("ack")),
					},
				},
				ChannelV2Genesis: channelv2types.DefaultGenesisState(),
			},
			expError: errors.New("invalid acknowledgement"),
		},
		{
			name: "invalid channel v2 genesis",
			genState: &types.GenesisState{
				ClientGenesis:     clienttypes.DefaultGenesisState(),
				ConnectionGenesis: connectiontypes.DefaultGenesisState(),
				ChannelGenesis:    channeltypes.DefaultGenesisState(),
				ChannelV2Genesis: channelv2types.GenesisState{
					Acknowledgements: []channelv2types.PacketState{
						channelv2types.NewPacketState(channel1, 1, nil),
					},
				},
			},
			expError: errors.New("invalid acknowledgement"),
		},
		{
			name: "invalid clientv2 genesis",
			genState: &types.GenesisState{
				ClientGenesis: clienttypes.DefaultGenesisState(),
				ClientV2Genesis: clientv2types.GenesisState{
					CounterpartyInfos: []clientv2types.GenesisCounterpartyInfo{
						{
							ClientId:         "",
							CounterpartyInfo: clientv2types.NewCounterpartyInfo([][]byte{{0o1}}, "test-0"),
						},
						{
							ClientId:         "test-0",
							CounterpartyInfo: clientv2types.NewCounterpartyInfo([][]byte{{0o1}}, "test-1"),
						},
					},
				},
				ConnectionGenesis: connectiontypes.DefaultGenesisState(),
				ChannelGenesis:    channeltypes.DefaultGenesisState(),
			},
			expError: errors.New("counterparty client id cannot be empty"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.genState.Validate()
		if tc.expError == nil {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
			suite.Require().Contains(err.Error(), tc.expError.Error())
		}
	}
}

func (suite *IBCTestSuite) TestInitGenesis() {
	header := suite.chainA.CreateTMClientHeader(suite.chainA.ChainID, suite.chainA.ProposedHeader.Height, clienttypes.NewHeight(0, uint64(suite.chainA.ProposedHeader.Height-1)), suite.chainA.ProposedHeader.Time, suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Vals, suite.chainA.Signers)

	testCases := []struct {
		name     string
		genState *types.GenesisState
	}{
		{
			name:     "default",
			genState: types.DefaultGenesisState(),
		},
		{
			name: "valid genesis",
			genState: &types.GenesisState{
				ClientGenesis: clienttypes.NewGenesisState(
					[]clienttypes.IdentifiedClientState{
						clienttypes.NewIdentifiedClientState(
							clientID, ibctm.NewClientState(suite.chainA.ChainID, ibctm.DefaultTrustLevel, ibctesting.TrustingPeriod, ibctesting.UnbondingPeriod, ibctesting.MaxClockDrift, clientHeight, commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath),
						),
					},
					[]clienttypes.ClientConsensusStates{
						clienttypes.NewClientConsensusStates(
							clientID,
							[]clienttypes.ConsensusStateWithHeight{
								clienttypes.NewConsensusStateWithHeight(
									header.GetHeight().(clienttypes.Height),
									ibctm.NewConsensusState(
										header.GetTime(), commitmenttypes.NewMerkleRoot(header.Header.AppHash), header.Header.NextValidatorsHash,
									),
								),
							},
						),
					},
					[]clienttypes.IdentifiedGenesisMetadata{
						clienttypes.NewIdentifiedGenesisMetadata(
							clientID,
							[]clienttypes.GenesisMetadata{
								clienttypes.NewGenesisMetadata([]byte("key1"), []byte("val1")),
								clienttypes.NewGenesisMetadata([]byte("key2"), []byte("val2")),
							},
						),
					},
					clienttypes.NewParams(exported.Tendermint),
					false,
					0,
				),
				ClientV2Genesis: clientv2types.GenesisState{
					CounterpartyInfos: []clientv2types.GenesisCounterpartyInfo{
						{
							ClientId:         "test-1",
							CounterpartyInfo: clientv2types.NewCounterpartyInfo([][]byte{{0o1}}, "test-0"),
						},
						{
							ClientId:         "test-0",
							CounterpartyInfo: clientv2types.NewCounterpartyInfo([][]byte{{0o1}}, "test-1"),
						},
					},
				},
				ConnectionGenesis: connectiontypes.NewGenesisState(
					[]connectiontypes.IdentifiedConnection{
						connectiontypes.NewIdentifiedConnection(connectionID, connectiontypes.NewConnectionEnd(connectiontypes.INIT, clientID, connectiontypes.NewCounterparty(clientID2, connectionID2, commitmenttypes.NewMerklePrefix([]byte("prefix"))), []*connectiontypes.Version{ibctesting.ConnectionVersion}, 0)),
					},
					[]connectiontypes.ConnectionPaths{
						connectiontypes.NewConnectionPaths(clientID, []string{connectionID}),
					},
					0,
					connectiontypes.NewParams(10),
				),
				ChannelGenesis: channeltypes.NewGenesisState(
					[]channeltypes.IdentifiedChannel{
						channeltypes.NewIdentifiedChannel(
							port1, channel1, channeltypes.NewChannel(
								channeltypes.INIT, channeltypes.ORDERED,
								channeltypes.NewCounterparty(port2, channel2), []string{connectionID}, ibctesting.DefaultChannelVersion,
							),
						),
					},
					[]channeltypes.PacketState{
						channeltypes.NewPacketState(port2, channel2, 1, []byte("ack")),
					},
					[]channeltypes.PacketState{
						channeltypes.NewPacketState(port2, channel2, 1, []byte("")),
					},
					[]channeltypes.PacketState{
						channeltypes.NewPacketState(port1, channel1, 1, []byte("commit_hash")),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port1, channel1, 1),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port2, channel2, 1),
					},
					[]channeltypes.PacketSequence{
						channeltypes.NewPacketSequence(port2, channel2, 1),
					},
					0,
					channeltypes.Params{UpgradeTimeout: channeltypes.DefaultTimeout},
				),
				ChannelV2Genesis: channelv2types.NewGenesisState(
					[]channelv2types.PacketState{
						channelv2types.NewPacketState(channel2, 1, []byte("ack")),
					},
					[]channelv2types.PacketState{
						channelv2types.NewPacketState(channel2, 1, []byte("")),
					},
					[]channelv2types.PacketState{
						channelv2types.NewPacketState(channel1, 1, []byte("commit_hash")),
					},
					[]channelv2types.PacketSequence{
						channelv2types.NewPacketSequence(channel1, 1),
					},
				),
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		app := simapp.Setup(suite.T(), false)

		suite.NotPanics(func() {
			ibc.InitGenesis(app.BaseApp.NewContext(false), *app.IBCKeeper, tc.genState)
		})
	}
}

func (suite *IBCTestSuite) TestExportGenesis() {
	testCases := []struct {
		msg      string
		malleate func()
	}{
		{
			"success",
			func() {
				// creates clients
				ibctesting.NewPath(suite.chainA, suite.chainB).Setup()
				// create extra clients
				ibctesting.NewPath(suite.chainA, suite.chainB).SetupClients()
				ibctesting.NewPath(suite.chainA, suite.chainB).SetupClients()
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest()

			tc.malleate()

			var gs *types.GenesisState
			suite.NotPanics(func() {
				gs = ibc.ExportGenesis(suite.chainA.GetContext(), *suite.chainA.App.GetIBCKeeper())
			})

			// init genesis based on export
			suite.NotPanics(func() {
				ibc.InitGenesis(suite.chainA.GetContext(), *suite.chainA.App.GetIBCKeeper(), gs)
			})

			suite.NotPanics(func() {
				cdc := codec.NewProtoCodec(suite.chainA.GetSimApp().InterfaceRegistry())
				genState := cdc.MustMarshalJSON(gs)
				cdc.MustUnmarshalJSON(genState, gs)
			})

			// init genesis based on marshal and unmarshal
			suite.NotPanics(func() {
				ibc.InitGenesis(suite.chainA.GetContext(), *suite.chainA.App.GetIBCKeeper(), gs)
			})
		})
	}
}
