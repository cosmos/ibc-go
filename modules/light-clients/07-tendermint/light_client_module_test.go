package tendermint_test

import (
	"crypto/sha256"
	"fmt"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
)

var (
	tmClientID          = clienttypes.FormatClientIdentifier(exported.Tendermint, 100)
	solomachineClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 0)
)

func (suite *TendermintTestSuite) TestStatus() {
	var (
		path        *ibctesting.Path
		clientState *ibctm.ClientState
	)

	testCases := []struct {
		name      string
		malleate  func()
		expStatus exported.Status
	}{
		{
			"client is active",
			func() {},
			exported.Active,
		},
		{
			"client is frozen",
			func() {
				clientState.FrozenHeight = clienttypes.NewHeight(0, 1)
				path.EndpointA.SetClientState(clientState)
			},
			exported.Frozen,
		},
		{
			"client status without consensus state",
			func() {
				clientState.LatestHeight = clientState.LatestHeight.Increment().(clienttypes.Height)
				path.EndpointA.SetClientState(clientState)
			},
			exported.Expired,
		},
		{
			"client status is expired",
			func() {
				suite.coordinator.IncrementTimeBy(clientState.TrustingPeriod)
			},
			exported.Expired,
		},
		{
			"client state not found",
			func() {
				store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)
				store.Delete(host.ClientStateKey())
			},
			exported.Unknown,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(path.EndpointA.ClientID)
			suite.Require().True(found)

			clientState = path.EndpointA.GetClientState().(*ibctm.ClientState)

			tc.malleate()

			status := lightClientModule.Status(suite.chainA.GetContext(), path.EndpointA.ClientID)
			suite.Require().Equal(tc.expStatus, status)
		})

	}
}

func (suite *TendermintTestSuite) TestGetTimestampAtHeight() {
	var (
		path   *ibctesting.Path
		height exported.Height
	)
	expectedTimestamp := time.Unix(1, 0)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"failure: client state not found for height",
			func() {
				store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)
				store.Delete(host.ClientStateKey())
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"failure: consensus state not found for height",
			func() {
				clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
				height = clientState.LatestHeight.Increment()
			},
			clienttypes.ErrConsensusStateNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			clientState := path.EndpointA.GetClientState().(*ibctm.ClientState)
			height = clientState.LatestHeight

			// grab consensusState from store and update with a predefined timestamp
			consensusState := path.EndpointA.GetConsensusState(height)
			tmConsensusState, ok := consensusState.(*ibctm.ConsensusState)
			suite.Require().True(ok)

			tmConsensusState.Timestamp = expectedTimestamp
			path.EndpointA.SetConsensusState(tmConsensusState, height)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(path.EndpointA.ClientID)
			suite.Require().True(found)

			tc.malleate()

			timestamp, err := lightClientModule.TimestampAtHeight(suite.chainA.GetContext(), path.EndpointA.ClientID, height)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				expectedTimestamp := uint64(expectedTimestamp.UnixNano())
				suite.Require().Equal(expectedTimestamp, timestamp)
			} else {
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *TendermintTestSuite) TestInitialize() {
	var consensusState exported.ConsensusState
	var clientState exported.ClientState

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"valid consensus & client states",
			func() {},
			nil,
		},
		{
			"invalid client state",
			func() {
				clientState.(*ibctm.ClientState).ChainId = ""
			},
			ibctm.ErrInvalidChainID,
		},
		{
			"invalid client state: solomachine client state",
			func() {
				clientState = ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2).ClientState()
			},
			fmt.Errorf("failed to unmarshal client state bytes into client state"),
		},
		{
			"invalid consensus: consensus state is solomachine consensus",
			func() {
				consensusState = ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, "solomachine", "", 2).ConsensusState()
			},
			fmt.Errorf("failed to unmarshal consensus state bytes into consensus state"),
		},
		{
			"invalid consensus state",
			func() {
				consensusState = ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte(ibctm.SentinelRoot)), []byte("invalidNextValsHash"))
			},
			fmt.Errorf("next validators hash is invalid"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewPath(suite.chainA, suite.chainB)

			tmConfig, ok := path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig)
			suite.Require().True(ok)

			clientState = ibctm.NewClientState(
				path.EndpointA.Chain.ChainID,
				tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
				suite.chainA.LatestCommittedHeader.GetHeight().(clienttypes.Height), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath,
			)

			consensusState = ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte(ibctm.SentinelRoot)), suite.chainA.ProposedHeader.ValidatorsHash)

			clientID := suite.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(suite.chainA.GetContext(), clientState.ClientType())

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			clientStateBz := suite.chainA.Codec.MustMarshal(clientState)
			consStateBz := suite.chainA.Codec.MustMarshal(consensusState)

			err := lightClientModule.Initialize(suite.chainA.GetContext(), path.EndpointA.ClientID, clientStateBz, consStateBz)

			store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), path.EndpointA.ClientID)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err, "valid case returned an error")
				suite.Require().True(store.Has(host.ClientStateKey()))
				suite.Require().True(store.Has(host.ConsensusStateKey(suite.chainB.LatestCommittedHeader.GetHeight())))
			} else {
				suite.Require().ErrorContains(err, tc.expErr.Error())
				suite.Require().False(store.Has(host.ClientStateKey()))
				suite.Require().False(store.Has(host.ConsensusStateKey(suite.chainB.LatestCommittedHeader.GetHeight())))
			}
		})
	}
}

func (suite *TendermintTestSuite) TestRecoverClient() {
	var (
		subjectClientID, substituteClientID string
		subjectClientState                  exported.ClientState
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
			},
			nil,
		},
		{
			"cannot parse malformed substitute client ID",
			func() {
				substituteClientID = ibctesting.InvalidID
			},
			host.ErrInvalidID,
		},
		{
			"substitute client ID does not contain 07-tendermint prefix",
			func() {
				substituteClientID = solomachineClientID
			},
			clienttypes.ErrInvalidClientType,
		},
		{
			"cannot find subject client state",
			func() {
				subjectClientID = tmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"cannot find substitute client state",
			func() {
				substituteClientID = tmClientID
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			ctx := suite.chainA.GetContext()

			subjectPath := ibctesting.NewPath(suite.chainA, suite.chainB)
			subjectPath.SetupClients()
			subjectClientID = subjectPath.EndpointA.ClientID
			subjectClientState = suite.chainA.GetClientState(subjectClientID)

			substitutePath := ibctesting.NewPath(suite.chainA, suite.chainB)
			substitutePath.SetupClients()
			substituteClientID = substitutePath.EndpointA.ClientID

			tmClientState, ok := subjectClientState.(*ibctm.ClientState)
			suite.Require().True(ok)
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(ctx, subjectPath.EndpointA.ClientID, tmClientState)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(subjectClientID)
			suite.Require().True(found)

			tc.malleate()

			err := lightClientModule.RecoverClient(ctx, subjectClientID, substituteClientID)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				// assert that status of subject client is now Active
				clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, subjectClientID)
				tmClientState, ok := subjectPath.EndpointA.GetClientState().(*ibctm.ClientState)
				suite.Require().True(ok)
				suite.Require().Equal(exported.Active, tmClientState.Status(ctx, clientStore, suite.chainA.App.AppCodec()))
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *TendermintTestSuite) TestVerifyUpgradeAndUpdateState() {
	var (
		clientID                                              string
		path                                                  *ibctesting.Path
		upgradedClientState                                   exported.ClientState
		upgradedClientStateAny, upgradedConsensusStateAny     *codectypes.Any
		upgradedClientStateProof, upgradedConsensusStateProof []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				// upgrade height is at next block
				upgradeHeight := clienttypes.NewHeight(0, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				zeroedUpgradedClient := upgradedClientState.(*ibctm.ClientState).ZeroCustomFields()
				zeroedUpgradedClientAny, err := codectypes.NewAnyWithValue(zeroedUpgradedClient)
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(zeroedUpgradedClientAny))
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedConsensusStateAny))
				suite.Require().NoError(err)

				// commit upgrade store changes and update clients
				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				upgradedClientStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
			},
			nil,
		},
		{
			"cannot find client state",
			func() {
				clientID = tmClientID
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"upgraded client state is not for tendermint client state",
			func() {
				upgradedClientStateAny = &codectypes.Any{
					Value: []byte("invalid client state bytes"),
				}
			},
			clienttypes.ErrInvalidClient,
		},
		{
			"upgraded consensus state is not tendermint consensus state",
			func() {
				upgradedConsensusStateAny = &codectypes.Any{
					Value: []byte("invalid consensus state bytes"),
				}
			},
			clienttypes.ErrInvalidConsensus,
		},
		{
			"upgraded client state height is not greater than current height",
			func() {
				// upgrade height is at next block
				upgradeHeight := clienttypes.NewHeight(1, uint64(suite.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				zeroedUpgradedClient := upgradedClientState.(*ibctm.ClientState).ZeroCustomFields()
				zeroedUpgradedClientAny, err := codectypes.NewAnyWithValue(zeroedUpgradedClient)
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(zeroedUpgradedClientAny))
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(suite.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), suite.chainB.Codec.MustMarshal(upgradedConsensusStateAny))
				suite.Require().NoError(err)

				// change upgraded client state height to be lower than current client state height
				tmClient, ok := upgradedClientState.(*ibctm.ClientState)
				suite.Require().True(ok)

				newLatestheight, ok := path.EndpointA.GetClientLatestHeight().Decrement()
				suite.Require().True(ok)

				tmClient.LatestHeight, ok = newLatestheight.(clienttypes.Height)
				suite.Require().True(ok)
				upgradedClientStateAny, err = codectypes.NewAnyWithValue(tmClient)
				suite.Require().NoError(err)

				suite.coordinator.CommitBlock(suite.chainB)
				err = path.EndpointA.UpdateClient()
				suite.Require().NoError(err)

				upgradedClientStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
				upgradedConsensusStateProof, _ = suite.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
			},
			ibcerrors.ErrInvalidHeight,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = ibctesting.NewPath(suite.chainA, suite.chainB)
			path.SetupClients()

			clientID = path.EndpointA.ClientID
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			suite.Require().True(ok)
			revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

			newUnbondindPeriod := ubdPeriod + trustingPeriod
			newChainID, err := clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
			suite.Require().NoError(err)

			upgradedClientState = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, newUnbondindPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), upgradePath)
			upgradedClientStateAny, err = codectypes.NewAnyWithValue(upgradedClientState)
			suite.Require().NoError(err)

			nextValsHash := sha256.Sum256([]byte("new-nextValsHash"))
			upgradedConsensusState := ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("new-hash")), nextValsHash[:])

			upgradedConsensusStateAny, err = codectypes.NewAnyWithValue(upgradedConsensusState)
			suite.Require().NoError(err)

			tc.malleate()

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			err = lightClientModule.VerifyUpgradeAndUpdateState(
				suite.chainA.GetContext(),
				clientID,
				upgradedClientStateAny.Value,
				upgradedConsensusStateAny.Value,
				upgradedClientStateProof,
				upgradedConsensusStateProof,
			)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)

				expClientState := path.EndpointA.GetClientState()
				expClientStateBz := suite.chainA.Codec.MustMarshal(expClientState)
				suite.Require().Equal(upgradedClientStateAny.Value, expClientStateBz)

				expConsensusState := ibctm.NewConsensusState(upgradedConsensusState.Timestamp, commitmenttypes.NewMerkleRoot([]byte(ibctm.SentinelRoot)), upgradedConsensusState.NextValidatorsHash)
				expConsensusStateBz := suite.chainA.Codec.MustMarshal(expConsensusState)

				consensusStateBz := suite.chainA.Codec.MustMarshal(path.EndpointA.GetConsensusState(path.EndpointA.GetClientLatestHeight()))
				suite.Require().Equal(expConsensusStateBz, consensusStateBz)
			} else {
				suite.Require().Error(err)
				suite.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (suite *TendermintTestSuite) TestVerifyMembership() {
	var (
		testingpath      *ibctesting.Path
		delayTimePeriod  uint64
		delayBlockPeriod uint64
		err              error
		proofHeight      exported.Height
		proof            []byte
		path             exported.Path
		value            []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"successful ClientState verification",
			func() {
				// default proof construction uses ClientState
			},
			nil,
		},
		{
			"successful ConsensusState verification", func() {
				latestHeight := testingpath.EndpointB.GetClientLatestHeight()

				key := host.FullConsensusStateKey(testingpath.EndpointB.ClientID, latestHeight)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)

				consensusState := testingpath.EndpointB.GetConsensusState(latestHeight).(*ibctm.ConsensusState)
				value, err = suite.chainB.Codec.MarshalInterface(consensusState)
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"successful Connection verification", func() {
				key := host.ConnectionKey(testingpath.EndpointB.ConnectionID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)

				connection := testingpath.EndpointB.GetConnection()
				value, err = suite.chainB.Codec.Marshal(&connection)
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"successful Channel verification", func() {
				key := host.ChannelKey(testingpath.EndpointB.ChannelConfig.PortID, testingpath.EndpointB.ChannelID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)

				channel := testingpath.EndpointB.GetChannel()
				value, err = suite.chainB.Codec.Marshal(&channel)
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"successful PacketCommitment verification", func() {
				// send from chainB to chainA since we are proving chainB sent a packet
				sequence, err := testingpath.EndpointB.SendPacket(clienttypes.NewHeight(1, 100), 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// make packet commitment proof
				packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, testingpath.EndpointB.ChannelConfig.PortID, testingpath.EndpointB.ChannelID, testingpath.EndpointA.ChannelConfig.PortID, testingpath.EndpointA.ChannelID, clienttypes.NewHeight(1, 100), 0)
				key := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)

				value = channeltypes.CommitPacket(suite.chainA.App.GetIBCKeeper().Codec(), packet)
			}, nil,
		},
		{
			"successful Acknowledgement verification", func() {
				// send from chainA to chainB since we are proving chainB wrote an acknowledgement
				sequence, err := testingpath.EndpointA.SendPacket(clienttypes.NewHeight(1, 100), 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// write receipt and ack
				packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, testingpath.EndpointA.ChannelConfig.PortID, testingpath.EndpointA.ChannelID, testingpath.EndpointB.ChannelConfig.PortID, testingpath.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)
				err = testingpath.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				key := host.PacketAcknowledgementKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)

				value = channeltypes.CommitAcknowledgement(ibcmock.MockAcknowledgement.Acknowledgement())
			},
			nil,
		},
		{
			"successful NextSequenceRecv verification", func() {
				// send from chainA to chainB since we are proving chainB incremented the sequence recv

				// send packet
				sequence, err := testingpath.EndpointA.SendPacket(clienttypes.NewHeight(1, 100), 0, ibctesting.MockPacketData)
				suite.Require().NoError(err)

				// next seq recv incremented
				packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, testingpath.EndpointA.ChannelConfig.PortID, testingpath.EndpointA.ChannelID, testingpath.EndpointB.ChannelConfig.PortID, testingpath.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)
				err = testingpath.EndpointB.RecvPacket(packet)
				suite.Require().NoError(err)

				key := host.NextSequenceRecvKey(packet.GetSourcePort(), packet.GetSourceChannel())
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)

				value = sdk.Uint64ToBigEndian(packet.GetSequence() + 1)
			},
			nil,
		},
		{
			"successful verification outside IBC store", func() {
				key := transfertypes.PortKey
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(commitmenttypes.NewMerklePrefix([]byte(transfertypes.StoreKey)), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProofForStore(transfertypes.StoreKey, key, int64(testingpath.EndpointA.GetClientLatestHeight().GetRevisionHeight()))

				value = []byte(suite.chainB.GetSimApp().TransferKeeper.GetPort(suite.chainB.GetContext()))
				suite.Require().NoError(err)
			},
			nil,
		},
		{
			"delay time period has passed", func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			nil,
		},
		{
			"delay time period has not passed", func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			ibctm.ErrDelayPeriodNotPassed,
		},
		{
			"delay block period has passed", func() {
				delayBlockPeriod = 1
			},
			nil,
		},
		{
			"delay block period has not passed", func() {
				delayBlockPeriod = 1000
			},
			ibctm.ErrDelayPeriodNotPassed,
		},
		{
			"latest client height < height", func() {
				proofHeight = testingpath.EndpointA.GetClientLatestHeight().Increment()
			},
			ibcerrors.ErrInvalidHeight,
		},
		{
			"invalid path type",
			func() {
				path = ibcmock.KeyPath{}
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"failed to unmarshal merkle proof", func() {
				proof = invalidProof
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"consensus state not found", func() {
				proofHeight = clienttypes.ZeroHeight()
			},
			clienttypes.ErrConsensusStateNotFound,
		},
		{
			"proof verification failed", func() {
				// change the value being proved
				value = []byte("invalid value")
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"proof is empty", func() {
				// change the inserted proof
				proof = []byte{}
			},
			commitmenttypes.ErrInvalidMerkleProof,
		},
		{
			"client state not found for height",
			func() {
				store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), testingpath.EndpointA.ClientID)
				store.Delete(host.ClientStateKey())
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			testingpath = ibctesting.NewPath(suite.chainA, suite.chainB)
			testingpath.SetChannelOrdered()
			testingpath.Setup()

			// reset time and block delays to 0, malleate may change to a specific non-zero value.
			delayTimePeriod = 0
			delayBlockPeriod = 0

			// create default proof, merklePath, and value which passes
			// may be overwritten by malleate()
			key := host.FullClientStateKey(testingpath.EndpointB.ClientID)
			merklePath := commitmenttypes.NewMerklePath(string(key))
			path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
			suite.Require().NoError(err)

			proof, proofHeight = suite.chainB.QueryProof(key)

			clientState := testingpath.EndpointB.GetClientState().(*ibctm.ClientState)
			value, err = suite.chainB.Codec.MarshalInterface(clientState)
			suite.Require().NoError(err)

			tc.malleate() // make changes as necessary

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(testingpath.EndpointA.ClientID)
			suite.Require().True(found)

			err = lightClientModule.VerifyMembership(
				suite.chainA.GetContext(), testingpath.EndpointA.ClientID, proofHeight, delayTimePeriod, delayBlockPeriod,
				proof, path, value,
			)
			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}

func (suite *TendermintTestSuite) TestVerifyNonMembership() {
	var (
		testingpath         *ibctesting.Path
		delayTimePeriod     uint64
		delayBlockPeriod    uint64
		err                 error
		proofHeight         exported.Height
		path                exported.Path
		proof               []byte
		invalidClientID     = "09-tendermint"
		invalidConnectionID = "connection-100"
		invalidChannelID    = "channel-800"
		invalidPortID       = "invalid-port"
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"successful ClientState verification of non membership",
			func() {
				// default proof construction uses ClientState
			},
			nil,
		},
		{
			"successful ConsensusState verification of non membership", func() {
				key := host.FullConsensusStateKey(invalidClientID, testingpath.EndpointB.GetClientLatestHeight())
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)
			},
			nil,
		},
		{
			"successful Connection verification of non membership", func() {
				key := host.ConnectionKey(invalidConnectionID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)
			},
			nil,
		},
		{
			"successful Channel verification of non membership", func() {
				key := host.ChannelKey(testingpath.EndpointB.ChannelConfig.PortID, invalidChannelID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)
			},
			nil,
		},
		{
			"successful PacketCommitment verification of non membership", func() {
				// make packet commitment proof
				key := host.PacketCommitmentKey(invalidPortID, invalidChannelID, 1)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)
			},
			nil,
		},
		{
			"successful Acknowledgement verification of non membership", func() {
				key := host.PacketAcknowledgementKey(invalidPortID, invalidChannelID, 1)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)
			},
			nil,
		},
		{
			"successful NextSequenceRecv verification of non membership", func() {
				key := host.NextSequenceRecvKey(invalidPortID, invalidChannelID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)
			},
			nil,
		},
		{
			"successful verification of non membership outside IBC store", func() {
				key := []byte{0x08}
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(commitmenttypes.NewMerklePrefix([]byte(transfertypes.StoreKey)), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProofForStore(transfertypes.StoreKey, key, int64(testingpath.EndpointA.GetClientLatestHeight().GetRevisionHeight()))
			},
			nil,
		},
		{
			"delay time period has passed", func() {
				delayTimePeriod = uint64(time.Second.Nanoseconds())
			},
			nil,
		},
		{
			"delay time period has not passed", func() {
				delayTimePeriod = uint64(time.Hour.Nanoseconds())
			},
			ibctm.ErrDelayPeriodNotPassed,
		},
		{
			"delay block period has passed", func() {
				delayBlockPeriod = 1
			},
			nil,
		},
		{
			"delay block period has not passed", func() {
				delayBlockPeriod = 1000
			},
			ibctm.ErrDelayPeriodNotPassed,
		},
		{
			"latest client height < height", func() {
				proofHeight = testingpath.EndpointA.GetClientLatestHeight().Increment()
			},
			ibcerrors.ErrInvalidHeight,
		},
		{
			"invalid path type",
			func() {
				path = ibcmock.KeyPath{}
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"failed to unmarshal merkle proof", func() {
				proof = invalidProof
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"consensus state not found", func() {
				proofHeight = clienttypes.ZeroHeight()
			},
			clienttypes.ErrConsensusStateNotFound,
		},
		{
			"verify non membership fails as path exists", func() {
				// change the value being proved
				key := host.FullClientStateKey(testingpath.EndpointB.ClientID)
				merklePath := commitmenttypes.NewMerklePath(string(key))
				path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
				suite.Require().NoError(err)

				proof, proofHeight = suite.chainB.QueryProof(key)
			},
			commitmenttypes.ErrInvalidProof,
		},
		{
			"proof is empty", func() {
				// change the inserted proof
				proof = []byte{}
			},
			commitmenttypes.ErrInvalidMerkleProof,
		},
		{
			"client state not found for height",
			func() {
				store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), testingpath.EndpointA.ClientID)
				store.Delete(host.ClientStateKey())
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			testingpath = ibctesting.NewPath(suite.chainA, suite.chainB)
			testingpath.SetChannelOrdered()
			testingpath.Setup()

			// reset time and block delays to 0, malleate may change to a specific non-zero value.
			delayTimePeriod = 0
			delayBlockPeriod = 0

			// create default proof, merklePath, and value which passes
			// may be overwritten by malleate()
			key := host.FullClientStateKey("invalid-client-id")

			merklePath := commitmenttypes.NewMerklePath(string(key))
			path, err = commitmenttypes.ApplyPrefix(suite.chainB.GetPrefix(), merklePath)
			suite.Require().NoError(err)

			proof, proofHeight = suite.chainB.QueryProof(key)

			tc.malleate() // make changes as necessary

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(testingpath.EndpointA.ClientID)
			suite.Require().True(found)

			err = lightClientModule.VerifyNonMembership(
				suite.chainA.GetContext(), testingpath.EndpointA.ClientID, proofHeight, delayTimePeriod, delayBlockPeriod,
				proof, path,
			)

			expPass := tc.expErr == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}
