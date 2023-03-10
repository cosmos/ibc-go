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
	//ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)
func (suite *WasmTestSuite) TestStatus() {
	//var (
	//	path        *ibctesting.Path
	//	clientState *wasmtypes.ClientState
	//)

	testCases := []struct {
		name      string
		malleate  func()
		expStatus exported.Status
	}{
		{"client is active", func() {}, exported.Active},
		/*{"client is frozen", func() {
			clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
			path.EndpointA.SetClientState(clientState)
		}, exported.Frozen},
		{"client status without consensus state", func() {
			clientState.LatestHeight = clientState.LatestHeight.Increment().(clienttypes.Height)
			path.EndpointA.SetClientState(clientState)
		}, exported.Expired},
		{"client status is expired", func() {
			suite.coordinator.IncrementTimeBy(clientState.TrustingPeriod)
		}, exported.Expired},*/
	}

	for _, tc := range testCases {
		tc.malleate()

		status := suite.clientState.Status(suite.chainA.GetContext(), suite.store, suite.chainA.App.AppCodec())
		suite.Require().Equal(tc.expStatus, status)

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
			consensusState: &wasmtypes.ConsensusState{
				Data: []byte("ics10-consensus-state"),
				CodeId: suite.codeId,
				Timestamp: uint64(suite.now.UnixNano()),
				Root: &commitmenttypes.MerkleRoot{
					Hash: []byte{0},
				},
			},
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

func (suite *WasmTestSuite) TestVerifyMemership() {
	var (
		clientState exported.ClientState
		err    error
		height exported.Height
		path   exported.Path
		proof  []byte
		value  []byte
	)
	clientID := "07-tendermint-0"
	connectionID := "connection-0"
	channelID := "channel-0"
	portID := "transfer"
	pathPrefix := "ibc/"
	delayTimePeriod := uint64(0)
	delayBlockPeriod := uint64(0)

	clientStateSetup := func() {
		clientState = suite.clientState

		height = clienttypes.NewHeight(2000, 10)
		key := host.FullClientStateKey(clientID)
		merklePath := commitmenttypes.NewMerklePath(string(key))
		path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 
		suite.Require().NoError(err)

		proof = make([]byte, base64.StdEncoding.DecodedLen(len(suite.testData["client_state_proof"])))
		_, err = base64.StdEncoding.Decode(proof, []byte(suite.testData["client_state_proof"]))
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
				RevisionHeight: 36,
			},
			ProofSpecs: commitmenttypes.GetSDKSpecs(),
			UpgradePath: []string{"upgrade", "upgradedIBCState"},
			AllowUpdateAfterExpiry: false,
			AllowUpdateAfterMisbehaviour: false,
		})
		suite.Require().NoError(err)
	}

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		{
			"successful ClientState verification",
			func() {
				clientStateSetup()
			},
			true,
		},
		/*{
			"successful ConsensusState verification",
			func() {

				clientState = suite.clientState

				height = clienttypes.NewHeight(2000, 10)
				key := host.FullConsensusStatePath(clientID, height)
				merklePath := commitmenttypes.NewMerklePath(key)
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 
				suite.Require().NoError(err)

				proof = make([]byte, base64.StdEncoding.DecodedLen(len(suite.testData["consensus_state_proof"])))
				_, err = base64.StdEncoding.Decode(proof, []byte(suite.testData["consensus_state_proof"]))
				suite.Require().NoError(err)

				value, err = suite.chainA.Codec.MarshalInterface(&tmtypes.ConsensusState{

				})
				suite.Require().NoError(err)
			},
			true,
		},*/
		{
			"successful Connection verification",
			func() {

				clientState = suite.clientState

				height = clienttypes.NewHeight(2000, 16)
				key := host.ConnectionPath(connectionID)
				merklePath := commitmenttypes.NewMerklePath(key)
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 
				suite.Require().NoError(err)

				proof = make([]byte, base64.StdEncoding.DecodedLen(len(suite.testData["connection_proof_ack"])))
				_, err = base64.StdEncoding.Decode(proof, []byte(suite.testData["connection_proof_ack"]))
				suite.Require().NoError(err)

				value, err = suite.chainA.Codec.Marshal(&types.ConnectionEnd{
					ClientId: clientID,
					Counterparty: types.Counterparty{
						ClientId: "08-wasm-0",
						ConnectionId: connectionID,
						Prefix: suite.chainA.GetPrefix(),
					},
					DelayPeriod: 0,
					State: types.OPEN,
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

				height = clienttypes.NewHeight(2000, 23)
				key := host.ChannelPath(portID, channelID)
				merklePath := commitmenttypes.NewMerklePath(key)
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof = make([]byte, base64.StdEncoding.DecodedLen(len(suite.testData["channel_proof_try"])))
				_, err = base64.StdEncoding.Decode(proof, []byte(suite.testData["channel_proof_try"]))
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
		/*{
			"successful PacketCommitment verification",
			func() {

				clientState = suite.clientState

				data := make([]byte, base64.StdEncoding.DecodedLen(len(suite.testData["packet_commitment_data"])))
				_, err = base64.StdEncoding.Decode(data, []byte(suite.testData["packet_commitment_data"]))
				suite.Require().NoError(err)

				fmt.Println("Data: ", string(data))
				height = clienttypes.NewHeight(2000, 35)
				packet := channeltypes.NewPacket(
					data,
					1, portID, channelID, portID, channelID, clienttypes.NewHeight(0, 3000),
					0,
				)
				key := host.PacketCommitmentPath(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				merklePath := commitmenttypes.NewMerklePath(key)
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof = make([]byte, base64.StdEncoding.DecodedLen(len(suite.testData["packet_commitment_proof"])))
				_, err = base64.StdEncoding.Decode(proof, []byte(suite.testData["packet_commitment_proof"]))
				suite.Require().NoError(err)
				
				value = channeltypes.CommitPacket(suite.chainA.App.GetIBCKeeper().Codec(), packet)
			},
			true,
		},*/
		/*{
			"successful Acknowledgement verification",
			func() {

				clientState = suite.clientState

				data := make([]byte, base64.StdEncoding.DecodedLen(len(suite.testData["acknowledgement"])))
				_, err = base64.StdEncoding.Decode(data, []byte(suite.testData["acknowledgement"]))
				suite.Require().NoError(err)

				height = clienttypes.NewHeight(2000, 32)
				packet := channeltypes.NewPacket(
					data,
					//[]byte(suite.testData["packet_commitment_data"]),
					1, portID, channelID, portID, channelID, clienttypes.NewHeight(2000, 1025),
					1678305199550951251,
				)
				key := host.PacketAcknowledgementKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 

				proof = make([]byte, base64.StdEncoding.DecodedLen(len(suite.testData["acknowledgement_proof"])))
				_, err = base64.StdEncoding.Decode(proof, []byte(suite.testData["acknowledgement_proof"]))
				suite.Require().NoError(err)

				value = []byte(suite.testData["acknowledgement"])//data//channeltypes.CommitPacket(suite.chainA.App.GetIBCKeeper().Codec(), packet)
			},
			true,
		},*/
		{
			"delay time period has passed", func() {
				clientStateSetup()
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			true,
		},
		/*{
			"delay time period has not passed", func() {
				clientStateSetup()
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			false,
		},*/
		{
			"delay block period has passed", func() {
				clientStateSetup()
				delayBlockPeriod = 1
			},
			true,
		},
		/*{
			"delay block period has not passed", func() {
				clientStateSetup()
				delayBlockPeriod = 1000
			},
			false,
		},*/
		{
			"latest client height < height", func() {
				clientStateSetup()
				height = height.Increment()
			}, false,
		},
		/*{ // Panics
			"invalid path type",
			func() {
				clientStateSetup()
				path = ibcmock.KeyPath{}
			},
			false,
		},*/
		/*{ // Panics
			"failed to unmarshal merkle proof", func() {
				clientStateSetup()
				proof = []byte("invalid proof")
			}, false,
		},*/
		/*{ //Panics
			"consensus state not found", func() {
				clientStateSetup()
				height = clienttypes.ZeroHeight()
			}, false,
		},*/
		/*{ // Panics
			"proof verification failed", func() {
				clientStateSetup()
				// change the value being proved
				value = []byte("invalid value")
			}, false,
		},*/
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWithChannel() // reset
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
	)
	//clientID := "07-tendermint-0"
	//connectionID := "connection-0"
	//channelID := "channel-0"
	//portID := "transfer"
	pathPrefix := "ibc/"
	delayTimePeriod := uint64(0)
	delayBlockPeriod := uint64(0)

	clientStateSetup := func() {
		clientState = suite.clientState

		height = clienttypes.NewHeight(2000, 10)
		key := host.FullClientStateKey("invalid-client-id")
		merklePath := commitmenttypes.NewMerklePath(string(key))
		path = commitmenttypes.NewMerklePath(append([]string{pathPrefix}, merklePath.KeyPath...)...) 
		suite.Require().NoError(err)
		proof = make([]byte, base64.StdEncoding.DecodedLen(len(suite.testData["client_state_proof"])))
		_, err = base64.StdEncoding.Decode(proof, []byte(suite.testData["client_state_proof"]))
		suite.Require().NoError(err)
	}

	testCases := []struct {
		name    string
		setup   func()
		expPass bool
	}{
		/*{
			"successful ClientState verification of non membership",
			func() {
			},
			true,
		},*/
		// Once this first test passes, add most of the next 16 from tm test case
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupWithChannel() // reset
			clientStateSetup()
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


/*func (suite *WasmTestSuite) TestVerifyHeader() {
	var (
		clientMsg   exported.ClientMessage
		clientState *wasmtypes.ClientState
	)

	// test singlesig and multisig public keys
	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful header",
				func() {
					data, err := hex.DecodeString(suite.testData["header_a0"])
					suite.Require().NoError(err)
					clientMsg = &wasmtypes.Header{
						Data: data,
						Height: clienttypes.Height{
							RevisionNumber: 1,
							RevisionHeight: 2,
						},
					}
					println(wm.ClientID)
				},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.name, func() {
				tc.setup()

				clientState = &suite.clientState
				err := clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)

				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}*/

// func (suite *WasmTestSuite) TestUpdateStateOnMisbehaviour() {
// 	var (
// 		clientMsg   exported.ClientMessage
// 		clientState *wasm.ClientState
// 	)

// 	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
// 		testCases := []struct {
// 			name    string
// 			setup   func()
// 			expPass bool
// 		}{
// 			{
// 				"successful update",
// 				func() {
// 					data, err := hex.DecodeString(suite.testData["header_a0"])
// 					suite.Require().NoError(err)
// 					clientMsg = &wasm.Header{
// 						Data: data,
// 						Height: clienttypes.Height{
// 							RevisionNumber: 1,
// 							RevisionHeight: 2,
// 						},
// 					}
// 					clientState = &suite.clientState
// 					println(wm.ClientID)
// 				},
// 				true,
// 			},
// 		}

// 		for _, tc := range testCases {
// 			tc := tc
// 			suite.Run(tc.name, func() {
// 				tc.setup()

// 				if tc.expPass {
// 					fmt.Println(clientMsg)
// 					suite.Require().NotPanics(func() {
// 						clientState.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)
// 					})
// 				} else {
// 					suite.Require().Panics(func() {
// 						clientState.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)
// 					})
// 				}
// 			})
// 		}
// 	}
// }

/*func (suite *WasmTestSuite) TestUpdateState() {
	var (
		clientMsg   exported.ClientMessage
		clientState *types.ClientState
	)

	// test singlesig and multisig public keys
	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful update",
				func() {
					data, err := hex.DecodeString(suite.testData["header_a0"])
					suite.Require().NoError(err)
					clientMsg = &types.Header{
						Data: data,
						Height: clienttypes.Height{
							RevisionNumber: 1,
							RevisionHeight: 2,
						},
					}
					clientState = &suite.clientState
					println(wm.ClientID)
				},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc
			suite.Run(tc.name, func() {
				tc.setup()

				if tc.expPass {
					consensusHeights := clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)

					clientStateBz := suite.store.Get(host.ClientStateKey())
					suite.Require().NotEmpty(clientStateBz)

					newClientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, clientStateBz)

					suite.Require().Len(consensusHeights, 1)
					suite.Require().Equal(clienttypes.Height{
						RevisionNumber: 2000,
						RevisionHeight: 89,
					}, consensusHeights[0])
					suite.Require().Equal(consensusHeights[0], newClientState.(*types.ClientState).LatestHeight)
				} else {
					suite.Require().Panics(func() {
						clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)
					})
				}
			})
		}
	}
}*/


// TODO: uncomment when fisherman is merged
/*
func (suite *WasmTestSuite) TestVerifyMisbehaviour() {
	var (
		clientMsg   exported.ClientMessage
		clientState *wasm.ClientState
	)

	for _, wm := range []*ibctesting.Wasm{suite.wasm} {
		testCases := []struct {
			name    string
			setup   func()
			expPass bool
		}{
			{
				"successful misbehaviour verification",
				func() {
					data, err := hex.DecodeString(suite.testData["misbehaviour_a0"])
					suite.Require().NoError(err)
					clientMsg = &wasm.Misbehaviour{
						ClientId: wm.ClientID,
						Data:     data,
					}
					clientState = &suite.clientState
					println(wm.ClientID)
				},
				true,
			},
		}

		for _, tc := range testCases {
			tc := tc
			suite.Run(tc.name, func() {
				tc.setup()
				println(clientMsg, clientState)
				err := clientState.VerifyClientMessage(suite.chainA.GetContext(), suite.chainA.Codec, suite.store, clientMsg)

				if tc.expPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			})
		}
	}
}
*/
