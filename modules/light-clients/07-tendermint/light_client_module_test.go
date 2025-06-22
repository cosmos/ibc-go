package tendermint_test

import (
	"crypto/sha256"
	"errors"
	"time"

	errorsmod "cosmossdk.io/errors"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

var (
	tmClientID          = clienttypes.FormatClientIdentifier(exported.Tendermint, 100)
	solomachineClientID = clienttypes.FormatClientIdentifier(exported.Solomachine, 0)
)

func (s *TendermintTestSuite) TestInitialize() {
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
				clientState = ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2).ClientState()
			},
			errors.New("failed to unmarshal client state bytes into client state"),
		},
		{
			"invalid consensus: consensus state is solomachine consensus",
			func() {
				consensusState = ibctesting.NewSolomachine(s.T(), s.chainA.Codec, "solomachine", "", 2).ConsensusState()
			},
			errors.New("failed to unmarshal consensus state bytes into consensus state"),
		},
		{
			"invalid consensus state",
			func() {
				consensusState = ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte(ibctm.SentinelRoot)), []byte("invalidNextValsHash"))
			},
			errors.New("next validators hash is invalid"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			path := ibctesting.NewPath(s.chainA, s.chainB)

			tmConfig, ok := path.EndpointA.ClientConfig.(*ibctesting.TendermintConfig)
			s.Require().True(ok)

			clientState = ibctm.NewClientState(
				path.EndpointA.Chain.ChainID,
				tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
				s.chainA.LatestCommittedHeader.GetHeight().(clienttypes.Height), commitmenttypes.GetSDKSpecs(), ibctesting.UpgradePath,
			)

			consensusState = ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte(ibctm.SentinelRoot)), s.chainA.ProposedHeader.ValidatorsHash)

			clientID := s.chainA.App.GetIBCKeeper().ClientKeeper.GenerateClientIdentifier(s.chainA.GetContext(), clientState.ClientType())

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			tc.malleate()

			clientStateBz := s.chainA.Codec.MustMarshal(clientState)
			consStateBz := s.chainA.Codec.MustMarshal(consensusState)

			err = lightClientModule.Initialize(s.chainA.GetContext(), path.EndpointA.ClientID, clientStateBz, consStateBz)

			store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)

			if tc.expErr == nil {
				s.Require().NoError(err, "valid case returned an error")
				s.Require().True(store.Has(host.ClientStateKey()))
				s.Require().True(store.Has(host.ConsensusStateKey(s.chainB.LatestCommittedHeader.GetHeight())))
			} else {
				s.Require().ErrorContains(err, tc.expErr.Error())
				s.Require().False(store.Has(host.ClientStateKey()))
				s.Require().False(store.Has(host.ConsensusStateKey(s.chainB.LatestCommittedHeader.GetHeight())))
			}
		})
	}
}

func (s *TendermintTestSuite) TestVerifyClientMessage() {
	var path *ibctesting.Path

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
			"failure: client state not found",
			func() {
				store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
				store.Delete(host.ClientStateKey())
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().NoError(err)

			// ensure counterparty state is committed
			s.coordinator.CommitBlock(s.chainB)
			trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
			s.Require().True(ok)
			header, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
			s.Require().NoError(err)

			tc.malleate()

			err = lightClientModule.VerifyClientMessage(s.chainA.GetContext(), path.EndpointA.ClientID, header)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *TendermintTestSuite) TestCheckForMisbehaviourPanicsOnClientStateNotFound() {
	s.SetupTest()

	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupClients()

	lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
	s.Require().NoError(err)

	// ensure counterparty state is committed
	s.coordinator.CommitBlock(s.chainB)
	trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
	s.Require().True(ok)
	header, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
	s.Require().NoError(err)

	// delete client state
	store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
	store.Delete(host.ClientStateKey())

	s.Require().PanicsWithError(errorsmod.Wrap(clienttypes.ErrClientNotFound, path.EndpointA.ClientID).Error(),
		func() {
			lightClientModule.CheckForMisbehaviour(s.chainA.GetContext(), path.EndpointA.ClientID, header)
		},
	)
}

func (s *TendermintTestSuite) TestUpdateStateOnMisbehaviourPanicsOnClientStateNotFound() {
	s.SetupTest()

	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupClients()

	lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
	s.Require().NoError(err)

	// ensure counterparty state is committed
	s.coordinator.CommitBlock(s.chainB)
	trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
	s.Require().True(ok)
	header, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
	s.Require().NoError(err)

	// delete client state
	store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
	store.Delete(host.ClientStateKey())

	s.Require().PanicsWithError(
		errorsmod.Wrap(clienttypes.ErrClientNotFound, path.EndpointA.ClientID).Error(),
		func() {
			lightClientModule.UpdateStateOnMisbehaviour(s.chainA.GetContext(), path.EndpointA.ClientID, header)
		},
	)
}

func (s *TendermintTestSuite) TestUpdateStatePanicsOnClientStateNotFound() {
	s.SetupTest()

	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupClients()

	lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
	s.Require().NoError(err)

	// ensure counterparty state is committed
	s.coordinator.CommitBlock(s.chainB)
	trustedHeight, ok := path.EndpointA.GetClientLatestHeight().(clienttypes.Height)
	s.Require().True(ok)
	header, err := path.EndpointA.Counterparty.Chain.IBCClientHeader(path.EndpointA.Counterparty.Chain.LatestCommittedHeader, trustedHeight)
	s.Require().NoError(err)

	// delete client state
	store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
	store.Delete(host.ClientStateKey())

	s.Require().PanicsWithError(
		errorsmod.Wrap(clienttypes.ErrClientNotFound, path.EndpointA.ClientID).Error(),
		func() {
			lightClientModule.UpdateState(s.chainA.GetContext(), path.EndpointA.ClientID, header)
		},
	)
}

func (s *TendermintTestSuite) TestVerifyMembership() {
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
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = s.chainB.QueryProof(key)

				consensusState, ok := testingpath.EndpointB.GetConsensusState(latestHeight).(*ibctm.ConsensusState)
				s.Require().True(ok)
				value, err = s.chainB.Codec.MarshalInterface(consensusState)
				s.Require().NoError(err)
			},
			nil,
		},
		{
			"successful Connection verification", func() {
				key := host.ConnectionKey(testingpath.EndpointB.ConnectionID)
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = s.chainB.QueryProof(key)

				connection := testingpath.EndpointB.GetConnection()
				value, err = s.chainB.Codec.Marshal(&connection)
				s.Require().NoError(err)
			},
			nil,
		},
		{
			"successful Channel verification", func() {
				key := host.ChannelKey(testingpath.EndpointB.ChannelConfig.PortID, testingpath.EndpointB.ChannelID)
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = s.chainB.QueryProof(key)

				channel := testingpath.EndpointB.GetChannel()
				value, err = s.chainB.Codec.Marshal(&channel)
				s.Require().NoError(err)
			},
			nil,
		},
		{
			"successful PacketCommitment verification", func() {
				// send from chainB to chainA since we are proving chainB sent a packet
				sequence, err := testingpath.EndpointB.SendPacket(clienttypes.NewHeight(1, 100), 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// make packet commitment proof
				packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, testingpath.EndpointB.ChannelConfig.PortID, testingpath.EndpointB.ChannelID, testingpath.EndpointA.ChannelConfig.PortID, testingpath.EndpointA.ChannelID, clienttypes.NewHeight(1, 100), 0)
				key := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)

				value = channeltypes.CommitPacket(packet)
			}, nil,
		},
		{
			"successful Acknowledgement verification", func() {
				// send from chainA to chainB since we are proving chainB wrote an acknowledgement
				sequence, err := testingpath.EndpointA.SendPacket(clienttypes.NewHeight(1, 100), 0, ibctesting.MockPacketData)
				s.Require().NoError(err)

				// write receipt and ack
				packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, testingpath.EndpointA.ChannelConfig.PortID, testingpath.EndpointA.ChannelID, testingpath.EndpointB.ChannelConfig.PortID, testingpath.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)
				err = testingpath.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)

				key := host.PacketAcknowledgementKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
				merklePath := commitmenttypes.NewMerklePath(key)

				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

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
				s.Require().NoError(err)

				// next seq recv incremented
				packet := channeltypes.NewPacket(ibctesting.MockPacketData, sequence, testingpath.EndpointA.ChannelConfig.PortID, testingpath.EndpointA.ChannelID, testingpath.EndpointB.ChannelConfig.PortID, testingpath.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)
				err = testingpath.EndpointB.RecvPacket(packet)
				s.Require().NoError(err)
				key := host.NextSequenceRecvKey(packet.GetDestPort(), packet.GetDestChannel())
				merklePath := commitmenttypes.NewMerklePath(key)

				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)

				value = sdk.Uint64ToBigEndian(packet.GetSequence() + 1)
			},
			nil,
		},
		{
			"successful verification outside IBC store", func() {
				key := transfertypes.PortKey
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(commitmenttypes.NewMerklePrefix([]byte(transfertypes.StoreKey)), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = s.chainB.QueryProofForStore(transfertypes.StoreKey, key, int64(testingpath.EndpointA.GetClientLatestHeight().GetRevisionHeight()))

				value = []byte(s.chainB.GetSimApp().TransferKeeper.GetPort(s.chainB.GetContext()))
				s.Require().NoError(err)
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
			"client state not found",
			func() {
				store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), testingpath.EndpointA.ClientID)
				store.Delete(host.ClientStateKey())
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			testingpath = ibctesting.NewPath(s.chainA, s.chainB)
			testingpath.SetChannelOrdered()
			testingpath.Setup()

			// reset time and block delays to 0, malleate may change to a specific non-zero value.
			delayTimePeriod = 0
			delayBlockPeriod = 0

			// create default proof, merklePath, and value which passes
			// may be overwritten by malleate()
			key := host.FullClientStateKey(testingpath.EndpointB.ClientID)
			merklePath := commitmenttypes.NewMerklePath(key)
			path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
			s.Require().NoError(err)

			proof, proofHeight = s.chainB.QueryProof(key)

			clientState, ok := testingpath.EndpointB.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			value, err = s.chainB.Codec.MarshalInterface(clientState)
			s.Require().NoError(err)

			tc.malleate() // make changes as necessary

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), testingpath.EndpointA.ClientID)
			s.Require().NoError(err)

			err = lightClientModule.VerifyMembership(
				s.chainA.GetContext(), testingpath.EndpointA.ClientID, proofHeight, delayTimePeriod, delayBlockPeriod,
				proof, path, value,
			)
			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}

func (s *TendermintTestSuite) TestVerifyNonMembership() {
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
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = s.chainB.QueryProof(key)
			},
			nil,
		},
		{
			"successful Connection verification of non membership", func() {
				key := host.ConnectionKey(invalidConnectionID)
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = s.chainB.QueryProof(key)
			},
			nil,
		},
		{
			"successful Channel verification of non membership", func() {
				key := host.ChannelKey(testingpath.EndpointB.ChannelConfig.PortID, invalidChannelID)
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = s.chainB.QueryProof(key)
			},
			nil,
		},
		{
			"successful PacketCommitment verification of non membership", func() {
				// make packet commitment proof
				key := host.PacketCommitmentKey(invalidPortID, invalidChannelID, 1)
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)
			},
			nil,
		},
		{
			"successful Acknowledgement verification of non membership", func() {
				key := host.PacketAcknowledgementKey(invalidPortID, invalidChannelID, 1)
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)
			},
			nil,
		},
		{
			"successful NextSequenceRecv verification of non membership", func() {
				key := host.NextSequenceRecvKey(invalidPortID, invalidChannelID)
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = testingpath.EndpointB.QueryProof(key)
			},
			nil,
		},
		{
			"successful verification of non membership outside IBC store", func() {
				key := []byte{0x08}
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(commitmenttypes.NewMerklePrefix([]byte(transfertypes.StoreKey)), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = s.chainB.QueryProofForStore(transfertypes.StoreKey, key, int64(testingpath.EndpointA.GetClientLatestHeight().GetRevisionHeight()))
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
				merklePath := commitmenttypes.NewMerklePath(key)
				path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
				s.Require().NoError(err)

				proof, proofHeight = s.chainB.QueryProof(key)
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
			"client state not found",
			func() {
				store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), testingpath.EndpointA.ClientID)
				store.Delete(host.ClientStateKey())
			},
			clienttypes.ErrClientNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			testingpath = ibctesting.NewPath(s.chainA, s.chainB)
			testingpath.SetChannelOrdered()
			testingpath.Setup()

			// reset time and block delays to 0, malleate may change to a specific non-zero value.
			delayTimePeriod = 0
			delayBlockPeriod = 0

			// create default proof, merklePath, and value which passes
			// may be overwritten by malleate()
			key := host.FullClientStateKey("invalid-client-id")

			merklePath := commitmenttypes.NewMerklePath(key)
			path, err = commitmenttypes.ApplyPrefix(s.chainB.GetPrefix(), merklePath)
			s.Require().NoError(err)

			proof, proofHeight = s.chainB.QueryProof(key)

			tc.malleate() // make changes as necessary

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), testingpath.EndpointA.ClientID)
			s.Require().NoError(err)

			err = lightClientModule.VerifyNonMembership(
				s.chainA.GetContext(), testingpath.EndpointA.ClientID, proofHeight, delayTimePeriod, delayBlockPeriod,
				proof, path,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}

func (s *TendermintTestSuite) TestStatus() {
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
				newLatestHeight, ok := clientState.LatestHeight.Increment().(clienttypes.Height)
				s.Require().True(ok)
				clientState.LatestHeight = newLatestHeight
				path.EndpointA.SetClientState(clientState)
			},
			exported.Expired,
		},
		{
			"client status is expired",
			func() {
				s.coordinator.IncrementTimeBy(clientState.TrustingPeriod)
			},
			exported.Expired,
		},
		{
			"client state not found",
			func() {
				store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
				store.Delete(host.ClientStateKey())
			},
			exported.Unknown,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().NoError(err)

			var ok bool
			clientState, ok = path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)

			tc.malleate()

			status := lightClientModule.Status(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().Equal(tc.expStatus, status)
		})
	}
}

func (s *TendermintTestSuite) TestLatestHeight() {
	var (
		path   *ibctesting.Path
		height exported.Height
	)

	testCases := []struct {
		name      string
		malleate  func()
		expHeight exported.Height
	}{
		{
			"success",
			func() {},
			clienttypes.Height{RevisionNumber: 0x1, RevisionHeight: 0x4},
		},
		{
			"client state not found",
			func() {
				store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
				store.Delete(host.ClientStateKey())
			},
			clienttypes.ZeroHeight(),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().NoError(err)

			tc.malleate()

			height = lightClientModule.LatestHeight(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().Equal(tc.expHeight, height)
		})
	}
}

func (s *TendermintTestSuite) TestGetTimestampAtHeight() {
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
			"failure: client state not found",
			func() {
				store := s.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(s.chainA.GetContext(), path.EndpointA.ClientID)
				store.Delete(host.ClientStateKey())
			},
			clienttypes.ErrClientNotFound,
		},
		{
			"failure: consensus state not found for height",
			func() {
				clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
				s.Require().True(ok)
				height = clientState.LatestHeight.Increment()
			},
			clienttypes.ErrConsensusStateNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			height = clientState.LatestHeight

			// grab consensusState from store and update with a predefined timestamp
			consensusState := path.EndpointA.GetConsensusState(height)
			tmConsensusState, ok := consensusState.(*ibctm.ConsensusState)
			s.Require().True(ok)

			tmConsensusState.Timestamp = expectedTimestamp
			path.EndpointA.SetConsensusState(tmConsensusState, height)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), path.EndpointA.ClientID)
			s.Require().NoError(err)

			tc.malleate()

			timestamp, err := lightClientModule.TimestampAtHeight(s.chainA.GetContext(), path.EndpointA.ClientID, height)

			if tc.expErr == nil {
				s.Require().NoError(err)

				expectedTimestamp := uint64(expectedTimestamp.UnixNano())
				s.Require().Equal(expectedTimestamp, timestamp)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *TendermintTestSuite) TestRecoverClient() {
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
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			ctx := s.chainA.GetContext()

			subjectPath := ibctesting.NewPath(s.chainA, s.chainB)
			subjectPath.SetupClients()
			subjectClientID = subjectPath.EndpointA.ClientID
			subjectClientState = s.chainA.GetClientState(subjectClientID)

			substitutePath := ibctesting.NewPath(s.chainA, s.chainB)
			substitutePath.SetupClients()
			substituteClientID = substitutePath.EndpointA.ClientID

			tmClientState, ok := subjectClientState.(*ibctm.ClientState)
			s.Require().True(ok)
			tmClientState.FrozenHeight = tmClientState.LatestHeight
			s.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(ctx, subjectPath.EndpointA.ClientID, tmClientState)

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), subjectClientID)
			s.Require().NoError(err)

			tc.malleate()

			err = lightClientModule.RecoverClient(ctx, subjectClientID, substituteClientID)

			if tc.expErr == nil {
				s.Require().NoError(err)

				// assert that status of subject client is now Active
				lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), subjectClientID)
				s.Require().NoError(err)
				s.Require().Equal(lightClientModule.Status(s.chainA.GetContext(), subjectClientID), exported.Active)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *TendermintTestSuite) TestVerifyUpgradeAndUpdateState() {
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
				upgradeHeight := clienttypes.NewHeight(0, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				zeroedUpgradedClient := upgradedClientState.(*ibctm.ClientState).ZeroCustomFields()
				zeroedUpgradedClientAny, err := codectypes.NewAnyWithValue(zeroedUpgradedClient)
				s.Require().NoError(err)

				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(zeroedUpgradedClientAny))
				s.Require().NoError(err)

				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(upgradedConsensusStateAny))
				s.Require().NoError(err)

				// commit upgrade store changes and update clients
				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				upgradedClientStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
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
				upgradeHeight := clienttypes.NewHeight(1, uint64(s.chainB.GetContext().BlockHeight()+1))

				// zero custom fields and store in upgrade store
				zeroedUpgradedClient := upgradedClientState.(*ibctm.ClientState).ZeroCustomFields()
				zeroedUpgradedClientAny, err := codectypes.NewAnyWithValue(zeroedUpgradedClient)
				s.Require().NoError(err)

				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedClient(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(zeroedUpgradedClientAny))
				s.Require().NoError(err)

				err = s.chainB.GetSimApp().UpgradeKeeper.SetUpgradedConsensusState(s.chainB.GetContext(), int64(upgradeHeight.GetRevisionHeight()), s.chainB.Codec.MustMarshal(upgradedConsensusStateAny))
				s.Require().NoError(err)

				// change upgraded client state height to be lower than current client state height
				tmClient, ok := upgradedClientState.(*ibctm.ClientState)
				s.Require().True(ok)

				newLatestheight, ok := path.EndpointA.GetClientLatestHeight().Decrement()
				s.Require().True(ok)

				tmClient.LatestHeight, ok = newLatestheight.(clienttypes.Height)
				s.Require().True(ok)
				upgradedClientStateAny, err = codectypes.NewAnyWithValue(tmClient)
				s.Require().NoError(err)

				s.coordinator.CommitBlock(s.chainB)
				err = path.EndpointA.UpdateClient()
				s.Require().NoError(err)

				upgradedClientStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedClientKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
				upgradedConsensusStateProof, _ = s.chainB.QueryUpgradeProof(upgradetypes.UpgradedConsStateKey(int64(upgradeHeight.GetRevisionHeight())), path.EndpointA.GetClientLatestHeight().GetRevisionHeight())
			},
			ibcerrors.ErrInvalidHeight,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = ibctesting.NewPath(s.chainA, s.chainB)
			path.SetupClients()

			clientID = path.EndpointA.ClientID
			clientState, ok := path.EndpointA.GetClientState().(*ibctm.ClientState)
			s.Require().True(ok)
			revisionNumber := clienttypes.ParseChainID(clientState.ChainId)

			newUnbondindPeriod := ubdPeriod + trustingPeriod
			newChainID, err := clienttypes.SetRevisionNumber(clientState.ChainId, revisionNumber+1)
			s.Require().NoError(err)

			upgradedClientState = ibctm.NewClientState(newChainID, ibctm.DefaultTrustLevel, trustingPeriod, newUnbondindPeriod, maxClockDrift, clienttypes.NewHeight(revisionNumber+1, clientState.LatestHeight.GetRevisionHeight()+1), commitmenttypes.GetSDKSpecs(), upgradePath)
			upgradedClientStateAny, err = codectypes.NewAnyWithValue(upgradedClientState)
			s.Require().NoError(err)

			nextValsHash := sha256.Sum256([]byte("new-nextValsHash"))
			upgradedConsensusState := ibctm.NewConsensusState(time.Now(), commitmenttypes.NewMerkleRoot([]byte("new-hash")), nextValsHash[:])

			upgradedConsensusStateAny, err = codectypes.NewAnyWithValue(upgradedConsensusState)
			s.Require().NoError(err)

			tc.malleate()

			lightClientModule, err := s.chainA.App.GetIBCKeeper().ClientKeeper.Route(s.chainA.GetContext(), clientID)
			s.Require().NoError(err)

			err = lightClientModule.VerifyUpgradeAndUpdateState(
				s.chainA.GetContext(),
				clientID,
				upgradedClientStateAny.Value,
				upgradedConsensusStateAny.Value,
				upgradedClientStateProof,
				upgradedConsensusStateProof,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)

				expClientState := path.EndpointA.GetClientState()
				expClientStateBz := s.chainA.Codec.MustMarshal(expClientState)
				s.Require().Equal(upgradedClientStateAny.Value, expClientStateBz)

				expConsensusState := ibctm.NewConsensusState(upgradedConsensusState.Timestamp, commitmenttypes.NewMerkleRoot([]byte(ibctm.SentinelRoot)), upgradedConsensusState.NextValidatorsHash)
				expConsensusStateBz := s.chainA.Codec.MustMarshal(expConsensusState)

				consensusStateBz := s.chainA.Codec.MustMarshal(path.EndpointA.GetConsensusState(path.EndpointA.GetClientLatestHeight()))
				s.Require().Equal(expConsensusStateBz, consensusStateBz)
			} else {
				s.Require().Error(err)
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}
