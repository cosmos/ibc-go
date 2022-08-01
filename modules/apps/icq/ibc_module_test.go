package icq_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	tmprotostate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmstate "github.com/tendermint/tendermint/state"

	"github.com/cosmos/ibc-go/v3/modules/apps/icq/types"
	icqtypes "github.com/cosmos/ibc-go/v3/modules/apps/icq/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
)

var (
	// TODO: Cosmos-SDK ADR-28: Update crypto.AddressHash() when sdk uses address.Module()
	// https://github.com/cosmos/cosmos-sdk/issues/10225
	//
	// TestAccAddress defines a resuable bech32 address for testing purposes
	// TestAccAddress = icqtypes.GenerateAddress(sdk.AccAddress(crypto.AddressHash([]byte(icqtypes.ModuleName))), ibctesting.FirstConnectionID, TestPortID)

	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"

	// TestPortID defines a resuable port identifier for testing purposes
	//TestPortID, _ = icqtypes.NewControllerPortID(TestOwnerAddress)

	// TestVersion defines a resuable interchainaccounts version string for testing purposes
	TestVersion = "icq-1"

	TestQueryPath = "/store/params/key"
	TestQueryData = "icqhost/HostEnabled"
)

type InterchainQueriesTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func TestInterchainQuerySuite(t *testing.T) {
	suite.Run(t, new(InterchainQueriesTestSuite))
}

func (suite *InterchainQueriesTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

func NewICQPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = icqtypes.PortID
	path.EndpointB.ChannelConfig.PortID = icqtypes.PortID
	path.EndpointA.ChannelConfig.Order = channeltypes.UNORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.UNORDERED
	path.EndpointA.ChannelConfig.Version = TestVersion
	path.EndpointB.ChannelConfig.Version = TestVersion

	return path
}

// SetupICQPath invokes the InterchainAccounts entrypoint and subsequent channel handshake handlers
func SetupICQPath(path *ibctesting.Path) error {
	if err := path.EndpointA.ChanOpenInit(); err != nil {
		return err
	}

	if err := path.EndpointB.ChanOpenTry(); err != nil {
		return err
	}

	if err := path.EndpointA.ChanOpenAck(); err != nil {
		return err
	}

	if err := path.EndpointB.ChanOpenConfirm(); err != nil {
		return err
	}

	return nil
}

// Test initiating a ChanOpenInit using the host chain instead of the controller chain
// ChainA is the controller chain. ChainB is the host chain
func (suite *InterchainQueriesTestSuite) TestChanOpenInit() {
	suite.SetupTest() // reset
	path := NewICQPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	// use chainB (host) for ChanOpenInit
	msg := channeltypes.NewMsgChannelOpenInit(path.EndpointB.ChannelConfig.PortID, icqtypes.Version, channeltypes.UNORDERED, []string{path.EndpointB.ConnectionID}, path.EndpointA.ChannelConfig.PortID, icqtypes.ModuleName)
	handler := suite.chainB.GetSimApp().MsgServiceRouter().Handler(msg)
	_, err := handler(suite.chainB.GetContext(), msg)

	suite.Require().Error(err)
}

func (suite *InterchainQueriesTestSuite) TestOnChanOpenTry() {
	var (
		path    *ibctesting.Path
		channel *channeltypes.Channel
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{

		{
			"success", func() {}, true,
		},
		{
			"host submodule disabled", func() {
				suite.chainB.GetSimApp().ICQKeeper.SetParams(suite.chainB.GetContext(), icqtypes.NewParams(false, []string{}))
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = NewICQPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			path.EndpointB.ChannelID = ibctesting.FirstChannelID

			// default values
			counterparty := channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channel = &channeltypes.Channel{
				State:          channeltypes.TRYOPEN,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{path.EndpointB.ConnectionID},
				Version:        path.EndpointB.ChannelConfig.Version,
			}

			tc.malleate()

			// ensure channel on chainB is set in state
			suite.chainB.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, *channel)

			module, _, err := suite.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
			suite.Require().NoError(err)

			chanCap, err := suite.chainB.App.GetScopedIBCKeeper().NewCapability(suite.chainB.GetContext(), host.ChannelCapabilityPath(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
			suite.Require().NoError(err)

			cbs, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			version, err := cbs.OnChanOpenTry(suite.chainB.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, chanCap, channel.Counterparty, path.EndpointA.ChannelConfig.Version,
			)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				suite.Require().Equal("", version)
			}

		})
	}

}

// Test initiating a ChanOpenAck using the host chain instead of the controller chain
// ChainA is the controller chain. ChainB is the host chain
func (suite *InterchainQueriesTestSuite) TestChanOpenAck() {
	suite.SetupTest() // reset
	path := NewICQPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	err := path.EndpointB.ChanOpenTry()
	suite.Require().NoError(err)

	// chainA maliciously sets channel to TRYOPEN
	channel := channeltypes.NewChannel(channeltypes.TRYOPEN, channeltypes.UNORDERED, channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, TestVersion)
	suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)

	// commit state changes so proof can be created
	suite.chainA.NextBlock()

	path.EndpointB.UpdateClient()

	// query proof from ChainA
	channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	proofTry, proofHeight := path.EndpointA.Chain.QueryProof(channelKey)

	// use chainB (host) for ChanOpenAck
	msg := channeltypes.NewMsgChannelOpenAck(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, path.EndpointA.ChannelID, TestVersion, proofTry, proofHeight, icqtypes.ModuleName)
	handler := suite.chainB.GetSimApp().MsgServiceRouter().Handler(msg)
	_, err = handler(suite.chainB.GetContext(), msg)

	suite.Require().Error(err)
}

func (suite *InterchainQueriesTestSuite) TestOnChanOpenConfirm() {
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{

		{
			"success", func() {}, true,
		},
		{
			"host submodule disabled", func() {
				suite.chainB.GetSimApp().ICQKeeper.SetParams(suite.chainB.GetContext(), icqtypes.NewParams(false, []string{}))
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := NewICQPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			suite.Require().NoError(err)

			tc.malleate()

			module, _, err := suite.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
			suite.Require().NoError(err)

			cbs, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			err = cbs.OnChanOpenConfirm(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

		})
	}

}

// OnChanCloseInit on host (chainB)
func (suite *InterchainQueriesTestSuite) TestOnChanCloseInit() {
	path := NewICQPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	err := SetupICQPath(path)
	suite.Require().NoError(err)

	module, _, err := suite.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
	suite.Require().NoError(err)

	cbs, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(module)
	suite.Require().True(ok)

	err = cbs.OnChanCloseInit(
		suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
	)

	suite.Require().Error(err)
}

func (suite *InterchainQueriesTestSuite) TestOnChanCloseConfirm() {
	var (
		path *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{

		{
			"success", func() {}, true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = NewICQPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICQPath(path)
			suite.Require().NoError(err)

			tc.malleate() // malleate mutates test data
			module, _, err := suite.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
			suite.Require().NoError(err)

			cbs, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			err = cbs.OnChanCloseConfirm(
				suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

		})
	}
}

func (suite *InterchainQueriesTestSuite) TestOnRecvPacket() {
	var (
		packetData []byte
	)
	testCases := []struct {
		name          string
		malleate      func()
		expAckSuccess bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"host submodule disabled", func() {
				suite.chainB.GetSimApp().ICQKeeper.SetParams(suite.chainB.GetContext(), icqtypes.NewParams(false, []string{}))
			}, false,
		},
		{
			"icq OnRecvPacket fails - cannot unmarshal packet data", func() {
				packetData = []byte("invalid data")
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := NewICQPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)
			err := SetupICQPath(path)
			suite.Require().NoError(err)

			// build packet data
			requests := []abcitypes.RequestQuery{
				{
					Path:   TestQueryPath,
					Height: 0,
					Data:   []byte(TestQueryData),
					Prove:  false,
				},
			}

			bz, err := types.SerializeCosmosQuery(requests)
			suite.Require().NoError(err)
			icqPacketData := types.InterchainQueryPacketData{
				Data: bz,
			}
			packetData = icqPacketData.GetBytes()

			// build expected ack
			resps := make([]abcitypes.ResponseQuery, len(requests))
			for i, req := range requests {
				resp := suite.chainB.GetSimApp().Query(req)
				resps[i] = abcitypes.ResponseQuery{
					Code:   resp.Code,
					Index:  resp.Index,
					Key:    resp.Key,
					Value:  resp.Value,
					Height: resp.Height,
				}
			}
			bz, err = types.SerializeCosmosResponse(resps)
			suite.Require().NoError(err)

			icqack := icqtypes.InterchainQueryPacketAck{
				Data: bz,
			}
			expectedTxResponse, err := icqtypes.ModuleCdc.MarshalJSON(&icqack)
			suite.Require().NoError(err)

			expectedAck := channeltypes.NewResultAcknowledgement(expectedTxResponse)

			params := icqtypes.NewParams(true, []string{TestQueryPath})
			suite.chainB.GetSimApp().ICQKeeper.SetParams(suite.chainB.GetContext(), params)

			// malleate packetData for test cases
			tc.malleate()

			seq := uint64(1)
			packet := channeltypes.NewPacket(packetData, seq, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 100), 0)

			tc.malleate()

			module, _, err := suite.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
			suite.Require().NoError(err)

			cbs, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			ack := cbs.OnRecvPacket(suite.chainB.GetContext(), packet, nil)
			if tc.expAckSuccess {
				suite.Require().True(ack.Success())
				suite.Require().Equal(expectedAck, ack)
			} else {
				suite.Require().False(ack.Success())
			}

		})
	}

}

func (suite *InterchainQueriesTestSuite) TestOnAcknowledgementPacket() {

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"icq OnAcknowledgementPacket fails with ErrInvalidChannelFlow", func() {}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := NewICQPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICQPath(path)
			suite.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			module, _, err := suite.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
			suite.Require().NoError(err)

			cbs, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			packet := channeltypes.NewPacket(
				[]byte("empty packet data"),
				suite.chainA.SenderAccount.GetSequence(),
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			err = cbs.OnAcknowledgementPacket(suite.chainB.GetContext(), packet, []byte("ackBytes"), nil)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *InterchainQueriesTestSuite) TestOnTimeoutPacket() {

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"icq OnTimeoutPacket fails with ErrInvalidChannelFlow", func() {}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path := NewICQPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICQPath(path)
			suite.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), path.EndpointB.ChannelConfig.PortID)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			packet := channeltypes.NewPacket(
				[]byte("empty packet data"),
				suite.chainA.SenderAccount.GetSequence(),
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			err = cbs.OnTimeoutPacket(suite.chainA.GetContext(), packet, nil)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// The safety of including SDK MsgResponses in the acknowledgement rests
// on the inclusion of the abcitypes.ResponseDeliverTx.Data in the
// abcitypes.ResposneDeliverTx hash. If the abcitypes.ResponseDeliverTx.Data
// gets removed from consensus they must no longer be used in the packet
// acknowledgement.
//
// This test acts as an indicqtor that the abcitypes.ResponseDeliverTx.Data
// may no longer be deterministic.
func (suite *InterchainQueriesTestSuite) TestABCICodeDeterminism() {
	msgResponseBz, err := proto.Marshal(&channeltypes.MsgChannelOpenInitResponse{})
	suite.Require().NoError(err)

	msgData := &sdk.MsgData{
		MsgType: sdk.MsgTypeURL(&channeltypes.MsgChannelOpenInit{}),
		Data:    msgResponseBz,
	}

	txResponse, err := proto.Marshal(&sdk.TxMsgData{
		Data: []*sdk.MsgData{msgData},
	})
	suite.Require().NoError(err)

	deliverTx := abcitypes.ResponseDeliverTx{
		Data: txResponse,
	}
	responses := tmprotostate.ABCIResponses{
		DeliverTxs: []*abcitypes.ResponseDeliverTx{
			&deliverTx,
		},
	}

	differentMsgResponseBz, err := proto.Marshal(&channeltypes.MsgRecvPacketResponse{})
	suite.Require().NoError(err)

	differentMsgData := &sdk.MsgData{
		MsgType: sdk.MsgTypeURL(&channeltypes.MsgRecvPacket{}),
		Data:    differentMsgResponseBz,
	}

	differentTxResponse, err := proto.Marshal(&sdk.TxMsgData{
		Data: []*sdk.MsgData{differentMsgData},
	})
	suite.Require().NoError(err)

	differentDeliverTx := abcitypes.ResponseDeliverTx{
		Data: differentTxResponse,
	}

	differentResponses := tmprotostate.ABCIResponses{
		DeliverTxs: []*abcitypes.ResponseDeliverTx{
			&differentDeliverTx,
		},
	}

	hash := tmstate.ABCIResponsesResultsHash(&responses)
	differentHash := tmstate.ABCIResponsesResultsHash(&differentResponses)

	suite.Require().NotEqual(hash, differentHash)
}
