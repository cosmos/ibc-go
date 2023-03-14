package types_test

import (
	"encoding/base64"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v7/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	tmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	wasmtypes "github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)
func (suite *WasmTestSuite) TestStatus() {
	testCases := []struct {
		name      string
		malleate  func()
		expStatus exported.Status
	}{
		{"client is active", func() {}, exported.Active},
		{"client is frozen", func() {
			cs, err := base64.StdEncoding.DecodeString(suite.testData["client_state_frozen"])
			suite.Require().NoError(err)

			frozenClientState := wasmtypes.ClientState{
				Data: cs,
				CodeId: suite.codeId,
				LatestHeight: clienttypes.Height{
					RevisionNumber: 2000,
					RevisionHeight: 5,
				},
			}

			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, "08-wasm-0", &frozenClientState)
		}, exported.Frozen},
		{"client status without consensus state", func() {
			cs, err := base64.StdEncoding.DecodeString(suite.testData["client_state_no_consensus"])
			suite.Require().NoError(err)

			clientState := wasmtypes.ClientState{
				Data: cs,
				CodeId: suite.codeId,
				LatestHeight: clienttypes.Height{
					RevisionNumber: 2000,
					RevisionHeight: 36, // This doesn't matter, but the grandpa client state is set to this
				},
			}
			
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.ctx, "08-wasm-0", &clientState)
		}, exported.Expired},
		// ics10-grandpa client state doesn't have a trusting period, so this might be removed
		/*{"client status is expired", func() {
			suite.coordinator.IncrementTimeBy(clientState.TrustingPeriod)
		}, exported.Expired},*/
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWithChannel()
			tc.malleate()

			status := suite.clientState.Status(suite.ctx, suite.store, suite.chainA.App.AppCodec())
			suite.Require().Equal(tc.expStatus, status)
		})
	}
}

func (suite *WasmTestSuite) TestValidate() {
	testCases := []struct {
		name        string
		clientState *wasmtypes.ClientState
		expPass     bool
	}{
		{
			name:        "valid client",
			clientState: wasmtypes.NewClientState([]byte{0}, []byte{0}, clienttypes.Height{}),
			expPass:     true,
		},
		{
			name:        "nil data",
			clientState: wasmtypes.NewClientState(nil, []byte{0}, clienttypes.Height{}),
			expPass:     false,
		},
		{
			name:        "empty data",
			clientState: wasmtypes.NewClientState([]byte{}, []byte{0}, clienttypes.Height{}),
			expPass:     false,
		},
		{
			name:        "nil code id",
			clientState: wasmtypes.NewClientState([]byte{0}, nil, clienttypes.Height{}),
			expPass:     false,
		},
		{
			name:        "empty code id",
			clientState: wasmtypes.NewClientState([]byte{0}, []byte{}, clienttypes.Height{}),
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

func (suite *WasmTestSuite) TestInitialize() {
	testCases := []struct {
		name           string
		consensusState exported.ConsensusState
		expPass        bool
	}{
		{
			name:           "valid consensus",
			consensusState: &suite.consensusState,
			expPass:        true,
		},
		{
			name:           "invalid consensus: consensus state is solomachine consensus",
			consensusState: ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2).ConsensusState(),
			expPass:        false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			store := suite.store
			err := suite.clientState.Initialize(suite.ctx, suite.chainA.Codec, store, tc.consensusState)

			if tc.expPass {
				suite.Require().NoError(err, "valid case returned an error")
				suite.Require().True(store.Has(host.ClientStateKey()))
				suite.Require().True(store.Has(host.ConsensusStateKey(suite.clientState.GetLatestHeight())))
			} else {
				suite.Require().Error(err, "invalid case didn't return an error")
				suite.Require().False(store.Has(host.ClientStateKey()))
				suite.Require().False(store.Has(host.ConsensusStateKey(suite.clientState.GetLatestHeight())))
			}
		})
	}
}

func (suite *WasmTestSuite) TestVerifyMembership() {
	var (
		clientState exported.ClientState
		err    error
		height exported.Height
		path   exported.Path
		proof  []byte
		value  []byte
		delayTimePeriod uint64
		delayBlockPeriod uint64
	)
	clientID := "07-tendermint-0"
	connectionID := "connection-0"
	channelID := "channel-0"
	portID := "transfer"
	pathPrefix := "ibc/"

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			"successful ClientState verification",
			func() {
			},
			true,
		},
		{
			"successful Connection verification",
			func() {

				clientState = suite.clientState

				height = clienttypes.NewHeight(2000, 11)
				key := host.ConnectionPath(connectionID)
				merklePath := commitmenttypes.NewMerklePath(key)
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 
				suite.Require().NoError(err)

				proof, err = base64.StdEncoding.DecodeString(suite.testData["connection_proof_try"])
				suite.Require().NoError(err)

				value, err = suite.chainA.Codec.Marshal(&types.ConnectionEnd{
					ClientId: clientID,
					Counterparty: types.Counterparty{
						ClientId: "08-wasm-0",
						ConnectionId: connectionID,
						Prefix: suite.chainA.GetPrefix(),
					},
					DelayPeriod: 0,
					State: types.TRYOPEN,
					Versions: []*types.Version{types.DefaultIBCVersion},
				})
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful Channel verification",
			func() {

				clientState = suite.clientState

				height = clienttypes.NewHeight(2000, 20)
				key := host.ChannelPath(portID, channelID)
				merklePath := commitmenttypes.NewMerklePath(key)
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof, err = base64.StdEncoding.DecodeString(suite.testData["channel_proof_try"])
				suite.Require().NoError(err)

				value, err = suite.chainA.Codec.Marshal(&channeltypes.Channel{
					State: channeltypes.TRYOPEN,
					Ordering: channeltypes.UNORDERED,
					Counterparty: channeltypes.Counterparty{
						PortId: portID,
						ChannelId: channelID,
					},
					ConnectionHops: []string{connectionID},
					Version: "ics20-1",
				})
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful PacketCommitment verification",
			func() {
				clientState = suite.clientState

				data, err := base64.StdEncoding.DecodeString(suite.testData["packet_commitment_data"])
				suite.Require().NoError(err)

				height = clienttypes.NewHeight(2000, 32)
				packet := channeltypes.NewPacket(
					data,
					1, portID, channelID, portID, channelID, clienttypes.NewHeight(0, 3000),
					0,
				)
				key := host.PacketCommitmentPath(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				merklePath := commitmenttypes.NewMerklePath(key)
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof, err = base64.StdEncoding.DecodeString(suite.testData["packet_commitment_proof"])
				suite.Require().NoError(err)
				
				value = channeltypes.CommitPacket(suite.chainA.App.GetIBCKeeper().Codec(), packet)
			},
			true,
		},
		{
			"successful Acknowledgement verification",
			func() {
				clientState = suite.clientState

				data, err := base64.StdEncoding.DecodeString(suite.testData["ack_data"])
				suite.Require().NoError(err)

				height = clienttypes.NewHeight(2000, 29)
				packet := channeltypes.NewPacket(
					data,
					uint64(1), portID, channelID, portID, channelID, clienttypes.NewHeight(2000, 1022),
					1678733040575532477,
				)
				key := host.PacketAcknowledgementKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof, err = base64.StdEncoding.DecodeString(suite.testData["ack_proof"])
				suite.Require().NoError(err)

				value, err = base64.StdEncoding.DecodeString(suite.testData["ack"])
				suite.Require().NoError(err)
				value = channeltypes.CommitAcknowledgement(value)
			},
			true,
		},
		{
			"delay time period has passed", func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			true,
		},
		{
			"delay time period has not passed", func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			true,
		},
		{
			"delay block period has passed", func() {
				delayBlockPeriod = 1
			},
			true,
		},
		{
			"delay block period has not passed", func() {
				delayBlockPeriod = 1000
			},
			true,
		},
		{
			"latest client height < height", func() {
				height = height.Increment()
			}, false,
		},
		{
			"invalid path type",
			func() {
				path = ibcmock.KeyPath{}
			},
			false,
		},
		{
			"failed to unmarshal merkle proof", func() {
				proof = []byte("invalid proof")
			}, false,
		},
		{
			"consensus state not found", func() {
				height = clienttypes.ZeroHeight()
			}, false,
		},
		{
			"proof verification failed", func() {
				// change the value being proved
				value = []byte("invalid value")
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWithChannel() // reset
			clientState = suite.clientState

			delayTimePeriod = 0
			delayBlockPeriod = 0
			height = clienttypes.NewHeight(2000, 11)
			key := host.FullClientStateKey(clientID)
			merklePath := commitmenttypes.NewMerklePath(string(key))
			path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

			proof, err = base64.StdEncoding.DecodeString(suite.testData["client_state_proof"])
			suite.Require().NoError(err)

			value, err = suite.chainA.Codec.MarshalInterface(&tmtypes.ClientState{
				ChainId: "simd",
				TrustLevel: tmtypes.Fraction{
					Numerator: 1,
					Denominator: 3,
				},
				TrustingPeriod: time.Duration(time.Second * 64000),
				UnbondingPeriod: time.Duration(time.Second * 1814400),
				MaxClockDrift: time.Duration(time.Second * 15),
				FrozenHeight: clienttypes.Height{
					RevisionNumber: 0,
					RevisionHeight: 0,
				},
				LatestHeight: clienttypes.Height{
					RevisionNumber: 0,
					RevisionHeight: 46,
				},
				ProofSpecs: commitmenttypes.GetSDKSpecs(),
				UpgradePath: []string{"upgrade", "upgradedIBCState"},
				AllowUpdateAfterExpiry: false,
				AllowUpdateAfterMisbehaviour: false,
			})
			suite.Require().NoError(err)

			tc.setup()

			err = clientState.VerifyMembership(
				suite.ctx, suite.store, suite.chainA.Codec,
				height, delayTimePeriod, delayBlockPeriod,
				proof, path, value,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *WasmTestSuite) TestVerifyNonMembership() {
	var (
		clientState exported.ClientState
		err    error
		height exported.Height
		path   exported.Path
		proof  []byte
		delayTimePeriod uint64
		delayBlockPeriod uint64
	)
	clientID := "07-tendermint-0"
	portID := "transfer"
	invalidClientID := "09-tendermint-0"
	invalidConnectionID := "connection-100"
	invalidChannelID    := "channel-800"
	pathPrefix := "ibc/"

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			"successful ClientState verification of non membership",
			func() {
			},
			true,
		},
		{
			"successful ConsensusState verification of non membership", func() {
				height = clienttypes.NewHeight(2000, 11)
				key := host.FullConsensusStateKey(invalidClientID, suite.clientState.GetLatestHeight())
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof, err = base64.StdEncoding.DecodeString(suite.testData["client_state_proof"])
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful Connection verification of non membership", func() {
				height = clienttypes.NewHeight(2000, 11)
				key := host.ConnectionKey(invalidConnectionID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof, err = base64.StdEncoding.DecodeString(suite.testData["connection_proof_try"])
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful Channel verification of non membership", func() {
				height = clienttypes.NewHeight(2000, 20)
				key := host.ChannelKey(portID, invalidChannelID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof, err = base64.StdEncoding.DecodeString(suite.testData["channel_proof_try"])
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"successful PacketCommitment verification of non membership", func() {
				height = clienttypes.NewHeight(2000, 32)
				key := host.PacketCommitmentKey(portID, invalidChannelID, 1)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof, err = base64.StdEncoding.DecodeString(suite.testData["packet_commitment_proof"])
				suite.Require().NoError(err)
			}, true,
		},
		{
			"successful Acknowledgement verification of non membership", func() {
				height = clienttypes.NewHeight(2000, 29)
				key := host.PacketAcknowledgementKey(portID, invalidChannelID, 1)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof, err = base64.StdEncoding.DecodeString(suite.testData["ack_proof"])
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"delay time period has passed", func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			true,
		},
		{
			"delay time period has not passed", func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			true,
		},
		{
			"delay block period has passed", func() {
				delayBlockPeriod = 1
			},
			true,
		},
		{
			"delay block period has not passed", func() {
				delayBlockPeriod = 1000
			},
			true,
		},
		{
			"latest client height < height", func() {
				height = clientState.GetLatestHeight().Increment()
			}, false,
		},
		{
			"invalid path type",
			func() {
				path = ibcmock.KeyPath{}
			},
			false,
		},
		{
			"failed to unmarshal merkle proof", func() {
				proof = []byte("invalid proof")
			}, false,
		},
		{
			"consensus state not found", func() {
				height = clienttypes.ZeroHeight()
			}, false,
		},
		{
			"verify non membership fails as path exists", func() {
				height = clienttypes.NewHeight(2000, 11)
				// change the value being proved
				key := host.FullClientStateKey(clientID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof, err = base64.StdEncoding.DecodeString(suite.testData["client_state_proof"])
				suite.Require().NoError(err)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWithChannel() // reset

			clientState = suite.clientState
			delayTimePeriod = 0
			delayBlockPeriod = 0
			height = clienttypes.NewHeight(2000, 11)
			key := host.FullClientStateKey(invalidClientID)
			merklePath := commitmenttypes.NewMerklePath(string(key))
			path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 
			proof, err = base64.StdEncoding.DecodeString(suite.testData["client_state_proof"])
			suite.Require().NoError(err)
			tc.setup()

			err = clientState.VerifyNonMembership(
				suite.ctx, suite.store, suite.chainA.Codec,
				height, delayTimePeriod, delayBlockPeriod,
				proof, path,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}