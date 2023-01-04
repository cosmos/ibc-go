package types_test

import (
	"time"

	ics23 "github.com/confio/ics23/go"

	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v3/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	"github.com/cosmos/ibc-go/v3/modules/light-clients/01-dymint/types"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
	ibcmock "github.com/cosmos/ibc-go/v3/testing/mock"
)

const (
	testClientID     = "clientidone"
	testConnectionID = "connectionid"
	testPortID       = "testportid"
	testChannelID    = "testchannelid"
	testSequence     = 1

	// Do not change the length of these variables
	fiftyCharChainID    = "12345678901234567890123456789012345678901234567890"
	fiftyOneCharChainID = "123456789012345678901234567890123456789012345678901"
)

var (
	invalidProof = []byte("invalid proof")
)

func (suite *DymintTestSuite) TestStatus() {
	var (
		path                    *ibctesting.Path
		clientState             *types.ClientState
		dymintCounterpartyChain *ibctesting.TestChain
		endpoint                *ibctesting.Endpoint
	)

	testCases := []struct {
		name      string
		malleate  func()
		expStatus exported.Status
	}{
		{"client is active", func() {}, exported.Active},
		{"client is frozen", func() {
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			endpoint.SetClientState(clientState)
		}, exported.Frozen},
		{"client status without consensus state", func() {
			clientState.LatestHeight = clientState.LatestHeight.Increment().(clienttypes.Height)
			endpoint.SetClientState(clientState)
		}, exported.Expired},
		{"client status is expired", func() {
			suite.coordinator.IncrementTimeBy(clientState.TrustingPeriod)
		}, exported.Expired},
	}

	for _, tc := range testCases {
		path = ibctesting.NewPath(suite.chainA, suite.chainB)
		suite.coordinator.SetupClients(path)

		if suite.chainB.TestChainClient.GetSelfClientType() == exported.Tendermint {
			// chainA must be Dymint
			dymintCounterpartyChain = suite.chainB
			endpoint = path.EndpointB
		} else {
			// chainB must be Dymint
			dymintCounterpartyChain = suite.chainA
			endpoint = path.EndpointA
		}

		clientStore := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), endpoint.ClientID)
		clientState = endpoint.GetClientState().(*types.ClientState)

		tc.malleate()

		status := clientState.Status(dymintCounterpartyChain.GetContext(), clientStore, dymintCounterpartyChain.App.AppCodec())
		suite.Require().Equal(tc.expStatus, status)

	}
}

func (suite *DymintTestSuite) TestValidate() {
	testCases := []struct {
		name        string
		clientState *types.ClientState
		expPass     bool
	}{
		{
			name:        "valid client",
			clientState: types.NewClientState(chainID, trustingPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expPass:     true,
		},
		{
			name:        "valid client with nil upgrade path",
			clientState: types.NewClientState(chainID, trustingPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), nil),
			expPass:     true,
		},
		{
			name:        "invalid chainID",
			clientState: types.NewClientState("  ", trustingPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expPass:     false,
		},
		{
			// NOTE: if this test fails, the code must account for the change in chainID length across tendermint versions!
			// Do not only fix the test, fix the code!
			// https://github.com/cosmos/ibc-go/issues/177
			name:        "valid chainID - chainID validation failed for chainID of length 50! ",
			clientState: types.NewClientState(fiftyCharChainID, trustingPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expPass:     true,
		},
		{
			// NOTE: if this test fails, the code must account for the change in chainID length across tendermint versions!
			// Do not only fix the test, fix the code!
			// https://github.com/cosmos/ibc-go/issues/177
			name:        "invalid chainID - chainID validation did not fail for chainID of length 51! ",
			clientState: types.NewClientState(fiftyOneCharChainID, trustingPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expPass:     false,
		},
		{
			name:        "invalid trusting period",
			clientState: types.NewClientState(chainID, 0, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expPass:     false,
		},
		{
			name:        "invalid max clock drift",
			clientState: types.NewClientState(chainID, trustingPeriod, 0, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			expPass:     false,
		},
		{
			name:        "invalid revision number",
			clientState: types.NewClientState(chainID, trustingPeriod, maxClockDrift, clienttypes.NewHeight(1, 1), commitmenttypes.GetSDKSpecs(), upgradePath),
			expPass:     false,
		},
		{
			name:        "invalid revision height",
			clientState: types.NewClientState(chainID, trustingPeriod, maxClockDrift, clienttypes.ZeroHeight(), commitmenttypes.GetSDKSpecs(), upgradePath),
			expPass:     false,
		},
		{
			name:        "proof specs is nil",
			clientState: types.NewClientState(chainID, ubdPeriod, maxClockDrift, height, nil, upgradePath),
			expPass:     false,
		},
		{
			name:        "proof specs contains nil",
			clientState: types.NewClientState(chainID, ubdPeriod, maxClockDrift, height, []*ics23.ProofSpec{ics23.TendermintSpec, nil}, upgradePath),
			expPass:     false,
		},
	}

	for _, tc := range testCases {
		err := tc.clientState.Validate()
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}

func (suite *DymintTestSuite) TestInitialize() {
	var (
		dymintCounterpartyChain *ibctesting.TestChain
		endpoint                *ibctesting.Endpoint
	)
	testCases := []struct {
		name           string
		consensusState exported.ConsensusState
		expPass        bool
	}{
		{
			name:           "valid consensus",
			consensusState: &types.ConsensusState{},
			expPass:        true,
		},
		{
			name:           "invalid consensus: consensus state is solomachine consensus",
			consensusState: ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2).ConsensusState(),
			expPass:        false,
		},
	}

	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	if suite.chainB.TestChainClient.GetSelfClientType() == exported.Tendermint {
		// chainA must be Dymint
		dymintCounterpartyChain = suite.chainB
		endpoint = path.EndpointB
	} else {
		// chainB must be Dymint
		dymintCounterpartyChain = suite.chainA
		endpoint = path.EndpointA
	}
	err := endpoint.CreateClient()
	suite.Require().NoError(err)

	clientState := dymintCounterpartyChain.GetClientState(endpoint.ClientID)
	store := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), endpoint.ClientID)

	for _, tc := range testCases {
		err := clientState.Initialize(dymintCounterpartyChain.GetContext(), dymintCounterpartyChain.Codec, store, tc.consensusState)
		if tc.expPass {
			suite.Require().NoError(err, "valid case returned an error")
		} else {
			suite.Require().Error(err, "invalid case didn't return an error")
		}
	}
}

func (suite *DymintTestSuite) TestVerifyClientConsensusState() {
	testCases := []struct {
		name           string
		clientState    *types.ClientState
		consensusState *types.ConsensusState
		prefix         commitmenttypes.MerklePrefix
		proof          []byte
		expPass        bool
	}{
		// FIXME: uncomment
		// {
		// 	name:        "successful verification",
		// 	clientState: types.NewClientState(chainID, trustingPeriod, maxClockDrift, height,  commitmenttypes.GetSDKSpecs()),
		// 	consensusState: types.ConsensusState{
		// 		Root: commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()),
		// 	},
		// 	prefix:  commitmenttypes.NewMerklePrefix([]byte("ibc")),
		// 	expPass: true,
		// },
		{
			name:        "ApplyPrefix failed",
			clientState: types.NewClientState(chainID, trustingPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			consensusState: &types.ConsensusState{
				Root: commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()),
			},
			prefix:  commitmenttypes.MerklePrefix{},
			expPass: false,
		},
		{
			name:        "latest client height < height",
			clientState: types.NewClientState(chainID, trustingPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			consensusState: &types.ConsensusState{
				Root: commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()),
			},
			prefix:  commitmenttypes.NewMerklePrefix([]byte("ibc")),
			expPass: false,
		},
		{
			name:        "proof verification failed",
			clientState: types.NewClientState(chainID, trustingPeriod, maxClockDrift, height, commitmenttypes.GetSDKSpecs(), upgradePath),
			consensusState: &types.ConsensusState{
				Root:               commitmenttypes.NewMerkleRoot(suite.header.Header.GetAppHash()),
				NextValidatorsHash: suite.valsHash,
			},
			prefix:  commitmenttypes.NewMerklePrefix([]byte("ibc")),
			proof:   []byte{},
			expPass: false,
		},
	}

	for i, tc := range testCases {
		tc := tc

		err := tc.clientState.VerifyClientConsensusState(
			nil, suite.cdc, height, "chainA", tc.clientState.LatestHeight, tc.prefix, tc.proof, tc.consensusState,
		)

		if tc.expPass {
			suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
		} else {
			suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

// test verification of the connection on chainB being represented in the
// light client on chainA
func (suite *DymintTestSuite) TestVerifyConnectionState() {
	var (
		clientState                          *types.ClientState
		proof                                []byte
		proofHeight                          exported.Height
		prefix                               commitmenttypes.MerklePrefix
		dymintChain, dymintCounterpartyChain *ibctesting.TestChain
		endpoint1, endpoint2                 *ibctesting.Endpoint
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"successful verification", func() {}, true,
		},
		{
			"ApplyPrefix failed", func() {
				prefix = commitmenttypes.MerklePrefix{}
			}, false,
		},
		{
			"latest client height < height", func() {
				proofHeight = clientState.LatestHeight.Increment()
			}, false,
		},
		{
			"proof verification failed", func() {
				proof = invalidProof
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTestWithConsensusType(exported.Tendermint, exported.Dymint) // reset

			// setup testing conditions
			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			if suite.chainB.TestChainClient.GetSelfClientType() == exported.Dymint {
				dymintCounterpartyChain = suite.chainA
				dymintChain = suite.chainB
				endpoint1 = path.EndpointA
				endpoint2 = path.EndpointB
			} else {
				dymintCounterpartyChain = suite.chainB
				dymintChain = suite.chainA
				endpoint1 = path.EndpointB
				endpoint2 = path.EndpointA
			}

			connection := endpoint2.GetConnection()

			var ok bool
			clientStateI := dymintCounterpartyChain.GetClientState(endpoint1.ClientID)
			clientState, ok = clientStateI.(*types.ClientState)
			suite.Require().True(ok)

			prefix = dymintChain.GetPrefix()

			// make connection proof
			connectionKey := host.ConnectionKey(endpoint2.ConnectionID)
			proof, proofHeight = dymintChain.QueryProof(connectionKey)

			tc.malleate() // make changes as necessary

			store := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), endpoint1.ClientID)

			err := clientState.VerifyConnectionState(
				store, dymintCounterpartyChain.Codec, proofHeight, &prefix, proof, endpoint2.ConnectionID, connection,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// test verification of the channel on chainB being represented in the light
// client on chainA
func (suite *DymintTestSuite) TestVerifyChannelState() {
	var (
		clientState                          *types.ClientState
		proof                                []byte
		proofHeight                          exported.Height
		prefix                               commitmenttypes.MerklePrefix
		dymintChain, dymintCounterpartyChain *ibctesting.TestChain
		endpoint1, endpoint2                 *ibctesting.Endpoint
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"successful verification", func() {}, true,
		},
		{
			"ApplyPrefix failed", func() {
				prefix = commitmenttypes.MerklePrefix{}
			}, false,
		},
		{
			"latest client height < height", func() {
				proofHeight = clientState.LatestHeight.Increment()
			}, false,
		},
		{
			"proof verification failed", func() {
				proof = invalidProof
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTestWithConsensusType(exported.Tendermint, exported.Dymint) // reset

			// setup testing conditions
			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			if suite.chainB.TestChainClient.GetSelfClientType() == exported.Dymint {
				dymintCounterpartyChain = suite.chainA
				dymintChain = suite.chainB
				endpoint1 = path.EndpointA
				endpoint2 = path.EndpointB
			} else {
				dymintCounterpartyChain = suite.chainB
				dymintChain = suite.chainA
				endpoint1 = path.EndpointB
				endpoint2 = path.EndpointA
			}

			channel := endpoint2.GetChannel()

			var ok bool
			clientStateI := dymintCounterpartyChain.GetClientState(endpoint1.ClientID)
			clientState, ok = clientStateI.(*types.ClientState)
			suite.Require().True(ok)

			prefix = dymintChain.GetPrefix()

			// make channel proof
			channelKey := host.ChannelKey(endpoint2.ChannelConfig.PortID, endpoint2.ChannelID)
			proof, proofHeight = dymintChain.QueryProof(channelKey)

			tc.malleate() // make changes as necessary

			store := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(dymintCounterpartyChain.GetContext(), endpoint1.ClientID)

			err := clientState.VerifyChannelState(
				store, dymintCounterpartyChain.Codec, proofHeight, &prefix, proof,
				endpoint2.ChannelConfig.PortID, endpoint2.ChannelID, channel,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// test verification of the packet commitment on chainB being represented
// in the light client on chainA. A send from chainB to chainA is simulated.
func (suite *DymintTestSuite) TestVerifyPacketCommitment() {
	var (
		clientState                          *types.ClientState
		proof                                []byte
		delayTimePeriod                      uint64
		delayBlockPeriod                     uint64
		proofHeight                          exported.Height
		prefix                               commitmenttypes.MerklePrefix
		dymintChain, dymintCounterpartyChain *ibctesting.TestChain
		endpoint1, endpoint2                 *ibctesting.Endpoint
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"successful verification", func() {}, true,
		},
		{
			name: "delay time period has passed",
			malleate: func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			expPass: true,
		},
		{
			name: "delay time period has not passed",
			malleate: func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			expPass: false,
		},
		{
			name: "delay block period has passed",
			malleate: func() {
				delayBlockPeriod = 1
			},
			expPass: true,
		},
		{
			name: "delay block period has not passed",
			malleate: func() {
				delayBlockPeriod = 1000
			},
			expPass: false,
		},

		{
			"ApplyPrefix failed", func() {
				prefix = commitmenttypes.MerklePrefix{}
			}, false,
		},
		{
			"latest client height < height", func() {
				proofHeight = clientState.LatestHeight.Increment()
			}, false,
		},
		{
			"proof verification failed", func() {
				proof = invalidProof
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// setup testing conditions
			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			if suite.chainB.TestChainClient.GetSelfClientType() == exported.Dymint {
				dymintCounterpartyChain = suite.chainA
				dymintChain = suite.chainB
				endpoint1 = path.EndpointA
				endpoint2 = path.EndpointB
			} else {
				dymintCounterpartyChain = suite.chainB
				dymintChain = suite.chainA
				endpoint1 = path.EndpointB
				endpoint2 = path.EndpointA
			}

			packet := channeltypes.NewPacket(ibctesting.MockPacketData, 1, endpoint2.ChannelConfig.PortID, endpoint2.ChannelID, endpoint1.ChannelConfig.PortID, endpoint1.ChannelID, clienttypes.NewHeight(0, 100), 0)
			err := endpoint2.SendPacket(packet)
			suite.Require().NoError(err)

			var ok bool
			clientStateI := dymintCounterpartyChain.GetClientState(endpoint1.ClientID)
			clientState, ok = clientStateI.(*types.ClientState)
			suite.Require().True(ok)

			prefix = dymintChain.GetPrefix()

			// make packet commitment proof
			packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			proof, proofHeight = endpoint2.QueryProof(packetKey)

			// reset time and block delays to 0, malleate may change to a specific non-zero value.
			delayTimePeriod = 0
			delayBlockPeriod = 0
			tc.malleate() // make changes as necessary

			ctx := dymintCounterpartyChain.GetContext()
			store := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, endpoint1.ClientID)

			commitment := channeltypes.CommitPacket(dymintCounterpartyChain.App.GetIBCKeeper().Codec(), packet)
			err = clientState.VerifyPacketCommitment(
				ctx, store, dymintCounterpartyChain.Codec, proofHeight, delayTimePeriod, delayBlockPeriod, &prefix, proof,
				packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence(), commitment,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// test verification of the acknowledgement on chainB being represented
// in the light client on chainA. A send and ack from chainA to chainB
// is simulated.
func (suite *DymintTestSuite) TestVerifyPacketAcknowledgement() {
	var (
		clientState                          *types.ClientState
		proof                                []byte
		delayTimePeriod                      uint64
		delayBlockPeriod                     uint64
		proofHeight                          exported.Height
		prefix                               commitmenttypes.MerklePrefix
		dymintCounterpartyChain, dymintChain *ibctesting.TestChain
		endpoint1, endpoint2                 *ibctesting.Endpoint
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"successful verification", func() {}, true,
		},
		{
			name: "delay time period has passed",
			malleate: func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			expPass: true,
		},
		{
			name: "delay time period has not passed",
			malleate: func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			expPass: false,
		},
		{
			name: "delay block period has passed",
			malleate: func() {
				delayBlockPeriod = 1
			},
			expPass: true,
		},
		{
			name: "delay block period has not passed",
			malleate: func() {
				delayBlockPeriod = 10
			},
			expPass: false,
		},

		{
			"ApplyPrefix failed", func() {
				prefix = commitmenttypes.MerklePrefix{}
			}, false,
		},
		{
			"latest client height < height", func() {
				proofHeight = clientState.LatestHeight.Increment()
			}, false,
		},
		{
			"proof verification failed", func() {
				proof = invalidProof
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// setup testing conditions
			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			if suite.chainB.TestChainClient.GetSelfClientType() == exported.Dymint {
				dymintCounterpartyChain = suite.chainA
				dymintChain = suite.chainB
				endpoint1 = path.EndpointA
				endpoint2 = path.EndpointB
			} else {
				dymintCounterpartyChain = suite.chainB
				dymintChain = suite.chainA
				endpoint1 = path.EndpointB
				endpoint2 = path.EndpointA
			}

			packet := channeltypes.NewPacket(ibctesting.MockPacketData, 1, endpoint1.ChannelConfig.PortID, endpoint1.ChannelID, endpoint2.ChannelConfig.PortID, endpoint2.ChannelID, clienttypes.NewHeight(0, 100), 0)

			// send packet
			err := endpoint1.SendPacket(packet)
			suite.Require().NoError(err)

			// write receipt and ack
			err = endpoint2.RecvPacket(packet)
			suite.Require().NoError(err)

			var ok bool
			clientStateI := dymintCounterpartyChain.GetClientState(endpoint1.ClientID)
			clientState, ok = clientStateI.(*types.ClientState)
			suite.Require().True(ok)

			prefix = dymintChain.GetPrefix()

			// make packet acknowledgement proof
			acknowledgementKey := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			proof, proofHeight = dymintChain.QueryProof(acknowledgementKey)

			// reset time and block delays to 0, malleate may change to a specific non-zero value.
			delayTimePeriod = 0
			delayBlockPeriod = 0
			tc.malleate() // make changes as necessary

			ctx := dymintCounterpartyChain.GetContext()
			store := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, endpoint1.ClientID)

			err = clientState.VerifyPacketAcknowledgement(
				ctx, store, dymintCounterpartyChain.Codec, proofHeight, delayTimePeriod, delayBlockPeriod, &prefix, proof,
				packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(), ibcmock.MockAcknowledgement.Acknowledgement(),
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// test verification of the absent acknowledgement on chainB being represented
// in the light client on chainA. A send from chainB to chainA is simulated, but
// no receive.
func (suite *DymintTestSuite) TestVerifyPacketReceiptAbsence() {
	var (
		clientState                          *types.ClientState
		proof                                []byte
		delayTimePeriod                      uint64
		delayBlockPeriod                     uint64
		proofHeight                          exported.Height
		prefix                               commitmenttypes.MerklePrefix
		dymintChain, dymintCounterpartyChain *ibctesting.TestChain
		endpoint1, endpoint2                 *ibctesting.Endpoint
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"successful verification", func() {}, true,
		},
		{
			name: "delay time period has passed",
			malleate: func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			expPass: true,
		},
		{
			name: "delay time period has not passed",
			malleate: func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			expPass: false,
		},
		{
			name: "delay block period has passed",
			malleate: func() {
				delayBlockPeriod = 1
			},
			expPass: true,
		},
		{
			name: "delay block period has not passed",
			malleate: func() {
				delayBlockPeriod = 10
			},
			expPass: false,
		},

		{
			"ApplyPrefix failed", func() {
				prefix = commitmenttypes.MerklePrefix{}
			}, false,
		},
		{
			"latest client height < height", func() {
				proofHeight = clientState.LatestHeight.Increment()
			}, false,
		},
		{
			"proof verification failed", func() {
				proof = invalidProof
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// setup testing conditions
			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			suite.coordinator.Setup(path)

			if suite.chainB.TestChainClient.GetSelfClientType() == exported.Dymint {
				dymintCounterpartyChain = suite.chainA
				dymintChain = suite.chainB
				endpoint1 = path.EndpointA
				endpoint2 = path.EndpointB
			} else {
				dymintCounterpartyChain = suite.chainB
				dymintChain = suite.chainA
				endpoint1 = path.EndpointB
				endpoint2 = path.EndpointA
			}

			packet := channeltypes.NewPacket(ibctesting.MockPacketData, 1, endpoint1.ChannelConfig.PortID, endpoint1.ChannelID, endpoint2.ChannelConfig.PortID, endpoint2.ChannelID, clienttypes.NewHeight(0, 100), 0)

			// send packet, but no recv
			err := endpoint1.SendPacket(packet)
			suite.Require().NoError(err)

			var ok bool
			clientStateI := dymintCounterpartyChain.GetClientState(endpoint1.ClientID)
			clientState, ok = clientStateI.(*types.ClientState)
			suite.Require().True(ok)

			prefix = dymintChain.GetPrefix()

			// make packet receipt absence proof
			receiptKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
			proof, proofHeight = endpoint2.QueryProof(receiptKey)

			// reset time and block delays to 0, malleate may change to a specific non-zero value.
			delayTimePeriod = 0
			delayBlockPeriod = 0
			tc.malleate() // make changes as necessary

			ctx := dymintCounterpartyChain.GetContext()
			store := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, endpoint1.ClientID)

			err = clientState.VerifyPacketReceiptAbsence(
				ctx, store, dymintCounterpartyChain.Codec, proofHeight, delayTimePeriod, delayBlockPeriod, &prefix, proof,
				packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence(),
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// test verification of the next receive sequence on chainB being represented
// in the light client on chainA. A send and receive from chainB to chainA is
// simulated.
func (suite *DymintTestSuite) TestVerifyNextSeqRecv() {
	var (
		clientState                          *types.ClientState
		proof                                []byte
		delayTimePeriod                      uint64
		delayBlockPeriod                     uint64
		proofHeight                          exported.Height
		prefix                               commitmenttypes.MerklePrefix
		dymintChain, dymintCounterpartyChain *ibctesting.TestChain
		endpoint1, endpoint2                 *ibctesting.Endpoint
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"successful verification", func() {}, true,
		},
		{
			name: "delay time period has passed",
			malleate: func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			expPass: true,
		},
		{
			name: "delay time period has not passed",
			malleate: func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			expPass: false,
		},
		{
			name: "delay block period has passed",
			malleate: func() {
				delayBlockPeriod = 1
			},
			expPass: true,
		},
		{
			name: "delay block period has not passed",
			malleate: func() {
				delayBlockPeriod = 10
			},
			expPass: false,
		},

		{
			"ApplyPrefix failed", func() {
				prefix = commitmenttypes.MerklePrefix{}
			}, false,
		},
		{
			"latest client height < height", func() {
				proofHeight = clientState.LatestHeight.Increment()
			}, false,
		},
		{
			"proof verification failed", func() {
				proof = invalidProof
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// setup testing conditions
			path := ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetChannelOrdered()
			suite.coordinator.Setup(path)

			if suite.chainB.TestChainClient.GetSelfClientType() == exported.Dymint {
				dymintCounterpartyChain = suite.chainA
				dymintChain = suite.chainB
				endpoint1 = path.EndpointA
				endpoint2 = path.EndpointB
			} else {
				dymintCounterpartyChain = suite.chainB
				dymintChain = suite.chainA
				endpoint1 = path.EndpointB
				endpoint2 = path.EndpointA
			}

			packet := channeltypes.NewPacket(ibctesting.MockPacketData, 1, endpoint1.ChannelConfig.PortID, endpoint1.ChannelID, endpoint2.ChannelConfig.PortID, endpoint2.ChannelID, clienttypes.NewHeight(0, 100), 0)

			// send packet
			err := endpoint1.SendPacket(packet)
			suite.Require().NoError(err)

			// next seq recv incremented
			err = endpoint2.RecvPacket(packet)
			suite.Require().NoError(err)

			var ok bool
			clientStateI := dymintCounterpartyChain.GetClientState(endpoint1.ClientID)
			clientState, ok = clientStateI.(*types.ClientState)
			suite.Require().True(ok)

			prefix = dymintChain.GetPrefix()

			// make next seq recv proof
			nextSeqRecvKey := host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
			proof, proofHeight = dymintChain.QueryProof(nextSeqRecvKey)

			// reset time and block delays to 0, malleate may change to a specific non-zero value.
			delayTimePeriod = 0
			delayBlockPeriod = 0
			tc.malleate() // make changes as necessary

			ctx := dymintCounterpartyChain.GetContext()
			store := dymintCounterpartyChain.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, endpoint1.ClientID)

			err = clientState.VerifyNextSequenceRecv(
				ctx, store, dymintCounterpartyChain.Codec, proofHeight, delayTimePeriod, delayBlockPeriod, &prefix, proof,
				packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence()+1,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
