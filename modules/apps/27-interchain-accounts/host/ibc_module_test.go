package host_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	feetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

var (
	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"

	// TestPortID defines a reusable port identifier for testing purposes
	TestPortID, _ = icatypes.NewControllerPortID(TestOwnerAddress)

	// TestVersion defines a reusable interchainaccounts version string for testing purposes
	TestVersion = string(icatypes.ModuleCdc.MustMarshalJSON(&icatypes.Metadata{
		Version:                icatypes.Version,
		ControllerConnectionId: ibctesting.FirstConnectionID,
		HostConnectionId:       ibctesting.FirstConnectionID,
		Encoding:               icatypes.EncodingProtobuf,
		TxType:                 icatypes.TxTypeSDKMultiMsg,
	}))
)

type InterchainAccountsTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func TestICATestSuite(t *testing.T) {
	suite.Run(t, new(InterchainAccountsTestSuite))
}

func (s *InterchainAccountsTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

func NewICAPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointA.ChannelConfig.Version = TestVersion
	path.EndpointB.ChannelConfig.Version = TestVersion

	return path
}

func RegisterInterchainAccount(endpoint *ibctesting.Endpoint, owner string) error {
	portID, err := icatypes.NewControllerPortID(owner)
	if err != nil {
		return err
	}

	channelSequence := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(endpoint.Chain.GetContext())

	if err := endpoint.Chain.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(endpoint.Chain.GetContext(), endpoint.ConnectionID, owner, endpoint.ChannelConfig.Version); err != nil {
		return err
	}

	// commit state changes for proof verification
	endpoint.Chain.NextBlock()

	// update port/channel ids
	endpoint.ChannelID = channeltypes.FormatChannelIdentifier(channelSequence)
	endpoint.ChannelConfig.PortID = portID

	return nil
}

// SetupICAPath invokes the InterchainAccounts entrypoint and subsequent channel handshake handlers
func SetupICAPath(path *ibctesting.Path, owner string) error {
	if err := RegisterInterchainAccount(path.EndpointA, owner); err != nil {
		return err
	}

	if err := path.EndpointB.ChanOpenTry(); err != nil {
		return err
	}

	if err := path.EndpointA.ChanOpenAck(); err != nil {
		return err
	}

	return path.EndpointB.ChanOpenConfirm()
}

// Test initiating a ChanOpenInit using the host chain instead of the controller chain
// ChainA is the controller chain. ChainB is the host chain
func (s *InterchainAccountsTestSuite) TestChanOpenInit() {
	s.SetupTest() // reset
	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	// use chainB (host) for ChanOpenInit
	msg := channeltypes.NewMsgChannelOpenInit(path.EndpointB.ChannelConfig.PortID, icatypes.Version, channeltypes.ORDERED, []string{path.EndpointB.ConnectionID}, path.EndpointA.ChannelConfig.PortID, icatypes.ModuleName)
	handler := s.chainB.GetSimApp().MsgServiceRouter().Handler(msg)
	_, err := handler(s.chainB.GetContext(), msg)

	s.Require().Error(err)
}

func (s *InterchainAccountsTestSuite) TestOnChanOpenTry() {
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
			"account address generation is block dependent", func() {
				icaHostAccount := icatypes.GenerateAddress(s.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
				err := s.chainB.GetSimApp().BankKeeper.SendCoins(s.chainB.GetContext(), s.chainB.SenderAccount.GetAddress(), icaHostAccount, sdk.Coins{sdk.NewCoin("stake", sdkmath.NewInt(1))})
				s.Require().NoError(err)
				s.Require().True(s.chainB.GetSimApp().AccountKeeper.HasAccount(s.chainB.GetContext(), icaHostAccount))

				// ensure account registration is simulated in a separate block
				s.chainB.NextBlock()
			}, true,
		},
		{
			"host submodule disabled", func() {
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), types.NewParams(false, []string{}))
			}, false,
		},
		{
			"success: ICA auth module callback returns error", func() {
				// mock module callback should not be called on host side
				s.chainB.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenTry = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
					portID, channelID string, chanCap *capabilitytypes.Capability,
					counterparty channeltypes.Counterparty, counterpartyVersion string,
				) (string, error) {
					return "", fmt.Errorf("mock ica auth fails")
				}
			}, true,
		},
		{
			"ICA callback fails - invalid channel order", func() {
				channel.Ordering = channeltypes.UNORDERED
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
			s.Require().NoError(err)
			path.EndpointB.ChannelID = ibctesting.FirstChannelID

			// default values
			counterparty := channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			channel = &channeltypes.Channel{
				State:          channeltypes.TRYOPEN,
				Ordering:       channeltypes.ORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{path.EndpointB.ConnectionID},
				Version:        path.EndpointB.ChannelConfig.Version,
			}

			tc.malleate()

			// ensure channel on chainB is set in state
			s.chainB.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, *channel)

			module, _, err := s.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
			s.Require().NoError(err)

			chanCap, err := s.chainB.App.GetScopedIBCKeeper().NewCapability(s.chainB.GetContext(), host.ChannelCapabilityPath(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID))
			s.Require().NoError(err)

			cbs, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			version, err := cbs.OnChanOpenTry(s.chainB.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, chanCap, channel.Counterparty, path.EndpointA.ChannelConfig.Version,
			)

			if tc.expPass {
				s.Require().NoError(err)

				addr, exists := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), path.EndpointB.ConnectionID, counterparty.PortId)
				s.Require().True(exists)
				s.Require().NotNil(addr)
			} else {
				s.Require().Error(err)
				s.Require().Equal("", version)
			}
		})
	}
}

// Test initiating a ChanOpenAck using the host chain instead of the controller chain
// ChainA is the controller chain. ChainB is the host chain
func (s *InterchainAccountsTestSuite) TestChanOpenAck() {
	s.SetupTest() // reset
	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
	s.Require().NoError(err)

	err = path.EndpointB.ChanOpenTry()
	s.Require().NoError(err)

	// chainA maliciously sets channel to TRYOPEN
	channel := channeltypes.NewChannel(channeltypes.TRYOPEN, channeltypes.ORDERED, channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID), []string{path.EndpointA.ConnectionID}, TestVersion)
	s.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel)

	// commit state changes so proof can be created
	s.chainA.NextBlock()

	err = path.EndpointB.UpdateClient()
	s.Require().NoError(err)

	// query proof from ChainA
	channelKey := host.ChannelKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	proofTry, proofHeight := path.EndpointA.Chain.QueryProof(channelKey)

	// use chainB (host) for ChanOpenAck
	msg := channeltypes.NewMsgChannelOpenAck(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, path.EndpointA.ChannelID, TestVersion, proofTry, proofHeight, icatypes.ModuleName)
	handler := s.chainB.GetSimApp().MsgServiceRouter().Handler(msg)
	_, err = handler(s.chainB.GetContext(), msg)

	s.Require().Error(err)
}

func (s *InterchainAccountsTestSuite) TestOnChanOpenConfirm() {
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
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), types.NewParams(false, []string{}))
			}, false,
		},
		{
			"success: ICA auth module callback returns error", func() {
				// mock module callback should not be called on host side
				s.chainB.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenConfirm = func(
					ctx sdk.Context, portID, channelID string,
				) error {
					return fmt.Errorf("mock ica auth fails")
				}
			}, true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()
			path := NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			s.Require().NoError(err)

			tc.malleate()

			module, _, err := s.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
			s.Require().NoError(err)

			cbs, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			err = cbs.OnChanOpenConfirm(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// OnChanCloseInit on host (chainB)
func (s *InterchainAccountsTestSuite) TestOnChanCloseInit() {
	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	s.Require().NoError(err)

	module, _, err := s.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
	s.Require().NoError(err)

	cbs, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(module)
	s.Require().True(ok)

	err = cbs.OnChanCloseInit(
		s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
	)

	s.Require().Error(err)
}

func (s *InterchainAccountsTestSuite) TestOnChanCloseConfirm() {
	var path *ibctesting.Path

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
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			s.Require().NoError(err)

			tc.malleate() // malleate mutates test data
			module, _, err := s.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
			s.Require().NoError(err)

			cbs, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			err = cbs.OnChanCloseConfirm(
				s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *InterchainAccountsTestSuite) TestOnRecvPacket() {
	var packetData []byte
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
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), types.NewParams(false, []string{}))
			}, false,
		},
		{
			"success with ICA auth module callback failure", func() {
				s.chainB.GetSimApp().ICAAuthModule.IBCApp.OnRecvPacket = func(
					ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress,
				) exported.Acknowledgement {
					return channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed OnRecvPacket mock callback"))
				}
			}, true,
		},
		{
			"ICA OnRecvPacket fails - cannot unmarshal packet data", func() {
				packetData = []byte("invalid data")
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path := NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)
			err := SetupICAPath(path, TestOwnerAddress)
			s.Require().NoError(err)

			// send 100stake to interchain account wallet
			amount, _ := sdk.ParseCoinsNormalized("100stake")
			interchainAccountAddr, _ := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
			bankMsg := &banktypes.MsgSend{FromAddress: s.chainB.SenderAccount.GetAddress().String(), ToAddress: interchainAccountAddr, Amount: amount}

			_, err = s.chainB.SendMsgs(bankMsg)
			s.Require().NoError(err)

			// build packet data
			msg := &banktypes.MsgSend{
				FromAddress: interchainAccountAddr,
				ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
				Amount:      amount,
			}
			data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
			s.Require().NoError(err)

			icaPacketData := icatypes.InterchainAccountPacketData{
				Type: icatypes.EXECUTE_TX,
				Data: data,
			}
			packetData = icaPacketData.GetBytes()

			// build expected ack
			protoAny, err := codectypes.NewAnyWithValue(&banktypes.MsgSendResponse{})
			s.Require().NoError(err)

			expectedTxResponse, err := proto.Marshal(&sdk.TxMsgData{
				MsgResponses: []*codectypes.Any{protoAny},
			})
			s.Require().NoError(err)

			expectedAck := channeltypes.NewResultAcknowledgement(expectedTxResponse)

			params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
			s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)

			// malleate packetData for test cases
			tc.malleate()

			seq := uint64(1)
			packet := channeltypes.NewPacket(packetData, seq, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.NewHeight(0, 100), 0)

			tc.malleate()

			module, _, err := s.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
			s.Require().NoError(err)

			cbs, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			ack := cbs.OnRecvPacket(s.chainB.GetContext(), packet, nil)
			if tc.expAckSuccess {
				s.Require().True(ack.Success())
				s.Require().Equal(expectedAck, ack)
			} else {
				s.Require().False(ack.Success())
			}
		})
	}
}

func (s *InterchainAccountsTestSuite) TestOnAcknowledgementPacket() {
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"ICA OnAcknowledgementPacket fails with ErrInvalidChannelFlow", func() {}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path := NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			s.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			module, _, err := s.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID)
			s.Require().NoError(err)

			cbs, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			packet := channeltypes.NewPacket(
				[]byte("empty packet data"),
				s.chainA.SenderAccount.GetSequence(),
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			err = cbs.OnAcknowledgementPacket(s.chainB.GetContext(), packet, []byte("ackBytes"), nil)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *InterchainAccountsTestSuite) TestOnTimeoutPacket() {
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"ICA OnTimeoutPacket fails with ErrInvalidChannelFlow", func() {}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path := NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			s.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), path.EndpointB.ChannelConfig.PortID)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			packet := channeltypes.NewPacket(
				[]byte("empty packet data"),
				s.chainA.SenderAccount.GetSequence(),
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			err = cbs.OnTimeoutPacket(s.chainA.GetContext(), packet, nil)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *InterchainAccountsTestSuite) fundICAWallet(ctx sdk.Context, portID string, amount sdk.Coins) {
	interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(ctx, ibctesting.FirstConnectionID, portID)
	s.Require().True(found)

	msgBankSend := &banktypes.MsgSend{
		FromAddress: s.chainB.SenderAccount.GetAddress().String(),
		ToAddress:   interchainAccountAddr,
		Amount:      amount,
	}

	res, err := s.chainB.SendMsgs(msgBankSend)
	s.Require().NotEmpty(res)
	s.Require().NoError(err)
}

// TestControlAccountAfterChannelClose tests that a controller chain can control a registered interchain account after the currently active channel for that interchain account has been closed.
// A new channel will be opened for the controller portID. The interchain account address should remain unchanged.
func (s *InterchainAccountsTestSuite) TestControlAccountAfterChannelClose() {
	path := NewICAPath(s.chainA, s.chainB)

	// use a fee enabled version to cover unwrapping channel version code paths
	feeMetadata := feetypes.Metadata{
		FeeVersion: feetypes.Version,
		AppVersion: TestVersion,
	}

	feeICAVersion := string(feetypes.ModuleCdc.MustMarshalJSON(&feeMetadata))

	path.EndpointA.ChannelConfig.Version = feeICAVersion
	path.EndpointB.ChannelConfig.Version = feeICAVersion

	s.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	s.Require().NoError(err)

	// two sends will be performed, one after initial creation of the account and one after channel closure and reopening
	var (
		startingBal           = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000)))
		tokenAmt              = sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)))
		expBalAfterFirstSend  = startingBal.Sub(tokenAmt...)
		expBalAfterSecondSend = expBalAfterFirstSend.Sub(tokenAmt...)
	)

	// check that the account is working as expected
	s.fundICAWallet(s.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID, startingBal)
	interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
	s.Require().True(found)

	msg := &banktypes.MsgSend{
		FromAddress: interchainAccountAddr,
		ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
		Amount:      tokenAmt,
	}

	data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
	s.Require().NoError(err)

	icaPacketData := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: data,
	}

	params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
	s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)

	//nolint: staticcheck // SA1019: ibctesting.FirstConnectionID is deprecated: use path.EndpointA.ConnectionID instead. (staticcheck)
	_, err = s.chainA.GetSimApp().ICAControllerKeeper.SendTx(s.chainA.GetContext(), nil, ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, icaPacketData, ^uint64(0))
	s.Require().NoError(err)
	err = path.EndpointB.UpdateClient()
	s.Require().NoError(err)

	// relay the packet
	packetRelay := channeltypes.NewPacket(icaPacketData.GetBytes(), 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.ZeroHeight(), ^uint64(0))
	err = path.RelayPacket(packetRelay)
	s.Require().NoError(err) // relay committed

	// check that the ica balance is updated
	icaAddr, err := sdk.AccAddressFromBech32(interchainAccountAddr)
	s.Require().NoError(err)

	s.assertBalance(icaAddr, expBalAfterFirstSend)

	// close the channel
	err = path.EndpointA.SetChannelState(channeltypes.CLOSED)
	s.Require().NoError(err)
	err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
	s.Require().NoError(err)

	// open a new channel on the same port
	path.EndpointA.ChannelID = ""
	path.EndpointB.ChannelID = ""
	s.coordinator.CreateChannels(path)

	//nolint: staticcheck // SA1019: ibctesting.FirstConnectionID is deprecated: use path.EndpointA.ConnectionID instead. (staticcheck)
	_, err = s.chainA.GetSimApp().ICAControllerKeeper.SendTx(s.chainA.GetContext(), nil, ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID, icaPacketData, ^uint64(0))
	s.Require().NoError(err)
	err = path.EndpointB.UpdateClient()
	s.Require().NoError(err)

	// relay the packet
	packetRelay = channeltypes.NewPacket(icaPacketData.GetBytes(), 1, path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, clienttypes.ZeroHeight(), ^uint64(0))
	err = path.RelayPacket(packetRelay)
	s.Require().NoError(err) // relay committed

	s.assertBalance(icaAddr, expBalAfterSecondSend)
}

// assertBalance asserts that the provided address has exactly the expected balance.
// CONTRACT: the expected balance must only contain one coin denom.
func (s *InterchainAccountsTestSuite) assertBalance(addr sdk.AccAddress, expBalance sdk.Coins) {
	balance := s.chainB.GetSimApp().BankKeeper.GetBalance(s.chainB.GetContext(), addr, sdk.DefaultBondDenom)
	s.Require().Equal(expBalance[0], balance)
}
