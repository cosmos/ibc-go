package keeper_test

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (suite *KeeperTestSuite) TestOnRecvPacket() {
	var (
		path       *ibctesting.Path
		packetData []byte
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"interchain account successfully executes an arbitrary message type using the * (allow all message types) param",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				// Populate the gov keeper in advance with an active proposal
				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}

				proposalMsg, err := govv1.NewLegacyContent(testProposal, interchainAccountAddr)
				suite.Require().NoError(err)

				proposal, err := govv1.NewProposal([]sdk.Msg{proposalMsg}, govtypes.DefaultStartingProposalID, suite.chainA.GetContext().BlockTime(), suite.chainA.GetContext().BlockTime(), "test proposal", "title", "Description", sdk.AccAddress(interchainAccountAddr))
				suite.Require().NoError(err)

				suite.chainB.GetSimApp().GovKeeper.SetProposal(suite.chainB.GetContext(), proposal)
				suite.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(suite.chainB.GetContext(), proposal)

				msg := &govtypes.MsgVote{
					ProposalId: govtypes.DefaultStartingProposalID,
					Voter:      interchainAccountAddr,
					Option:     govtypes.OptionYes,
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{"*"})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes banktypes.MsgSend",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &banktypes.MsgSend{
					FromAddress: interchainAccountAddr,
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes stakingtypes.MsgDelegate",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				validatorAddr := (sdk.ValAddress)(suite.chainB.Vals.Validators[0].Address)
				msg := &stakingtypes.MsgDelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000)),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes stakingtypes.MsgDelegate and stakingtypes.MsgUndelegate sequentially",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				validatorAddr := (sdk.ValAddress)(suite.chainB.Vals.Validators[0].Address)
				msgDelegate := &stakingtypes.MsgDelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000)),
				}

				msgUndelegate := &stakingtypes.MsgUndelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000)),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msgDelegate, msgUndelegate})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msgDelegate), sdk.MsgTypeURL(msgUndelegate)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes govtypes.MsgSubmitProposal",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}

				protoAny, err := codectypes.NewAnyWithValue(testProposal)
				suite.Require().NoError(err)

				msg := &govtypes.MsgSubmitProposal{
					Content:        protoAny,
					InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000))),
					Proposer:       interchainAccountAddr,
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes govtypes.MsgVote",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				// Populate the gov keeper in advance with an active proposal
				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}

				proposalMsg, err := govv1.NewLegacyContent(testProposal, interchainAccountAddr)
				suite.Require().NoError(err)

				proposal, err := govv1.NewProposal([]sdk.Msg{proposalMsg}, govtypes.DefaultStartingProposalID, suite.chainA.GetContext().BlockTime(), suite.chainA.GetContext().BlockTime(), "test proposal", "title", "description", sdk.AccAddress(interchainAccountAddr))
				suite.Require().NoError(err)

				suite.chainB.GetSimApp().GovKeeper.SetProposal(suite.chainB.GetContext(), proposal)
				suite.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(suite.chainB.GetContext(), proposal)

				msg := &govtypes.MsgVote{
					ProposalId: govtypes.DefaultStartingProposalID,
					Voter:      interchainAccountAddr,
					Option:     govtypes.OptionYes,
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes disttypes.MsgFundCommunityPool",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &disttypes.MsgFundCommunityPool{
					Amount:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(5000))),
					Depositor: interchainAccountAddr,
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes disttypes.MsgSetWithdrawAddress",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &disttypes.MsgSetWithdrawAddress{
					DelegatorAddress: interchainAccountAddr,
					WithdrawAddress:  suite.chainB.SenderAccount.GetAddress().String(),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes transfertypes.MsgTransfer",
			func() {
				transferPath := ibctesting.NewPath(suite.chainB, suite.chainC)
				transferPath.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
				transferPath.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort
				transferPath.EndpointA.ChannelConfig.Version = transfertypes.Version
				transferPath.EndpointB.ChannelConfig.Version = transfertypes.Version

				suite.coordinator.Setup(transferPath)

				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &transfertypes.MsgTransfer{
					SourcePort:       transferPath.EndpointA.ChannelConfig.PortID,
					SourceChannel:    transferPath.EndpointA.ChannelID,
					Token:            sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100)),
					Sender:           interchainAccountAddr,
					Receiver:         suite.chainA.SenderAccount.GetAddress().String(),
					TimeoutHeight:    clienttypes.NewHeight(1, 100),
					TimeoutTimestamp: uint64(0),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"unregistered sdk.Msg",
			func() {
				msg := &banktypes.MsgSendResponse{}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{"/" + proto.MessageName(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			false,
		},
		{
			"cannot unmarshal interchain account packet data",
			func() {
				packetData = []byte{}
			},
			false,
		},
		{
			"cannot deserialize interchain account packet data messages",
			func() {
				data := []byte("invalid packet data")

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"invalid packet type - UNSPECIFIED",
			func() {
				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{&banktypes.MsgSend{}})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.UNSPECIFIED,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"unauthorised: interchain account not found for controller port ID",
			func() {
				path.EndpointA.ChannelConfig.PortID = "invalid-port-id"

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{&banktypes.MsgSend{}})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"unauthorised: message type not allowed", // NOTE: do not update params to explicitly force the error
			func() {
				msg := &banktypes.MsgSend{
					FromAddress: suite.chainB.SenderAccount.GetAddress().String(),
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"unauthorised: signer address is not the interchain account associated with the controller portID",
			func() {
				msg := &banktypes.MsgSend{
					FromAddress: suite.chainB.SenderAccount.GetAddress().String(), // unexpected signer
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg})
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, encoding := range testedEncodings {
		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.msg, func() {
				suite.SetupTest() // reset

				path = NewICAPath(suite.chainA, suite.chainB, encoding)
				suite.coordinator.SetupConnections(path)

				err := SetupICAPath(path, TestOwnerAddress)
				suite.Require().NoError(err)

				portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
				suite.Require().NoError(err)

				// Get the address of the interchain account stored in state during handshake step
				storedAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, portID)
				suite.Require().True(found)

				icaAddr, err := sdk.AccAddressFromBech32(storedAddr)
				suite.Require().NoError(err)

				// Check if account is created
				interchainAccount := suite.chainB.GetSimApp().AccountKeeper.GetAccount(suite.chainB.GetContext(), icaAddr)
				suite.Require().Equal(interchainAccount.GetAddress().String(), storedAddr)

				suite.fundICAWallet(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1000000))))

				tc.malleate(encoding) // malleate mutates test data

				packet := channeltypes.NewPacket(
					packetData,
					suite.chainA.SenderAccount.GetSequence(),
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					path.EndpointB.ChannelConfig.PortID,
					path.EndpointB.ChannelID,
					suite.chainB.GetTimeoutHeight(),
					0,
				)

				txResponse, err := suite.chainB.GetSimApp().ICAHostKeeper.OnRecvPacket(suite.chainB.GetContext(), packet)

				expPass := tc.expErr == nil
				if expPass {
					suite.Require().NoError(err)
					suite.Require().NotNil(txResponse)
				} else {
					suite.Require().ErrorIs(err, tc.expErr)
					suite.Require().Nil(txResponse)
				}
			})
		}
	}
}

func (suite *KeeperTestSuite) TestJSONOnRecvPacket() {
	var (
		path       *ibctesting.Path
		packetData []byte
	)
	interchainAccountAddr := "cosmos15ulrf36d4wdtrtqzkgaan9ylwuhs7k7qz753uk"

	testCases := []struct {
		msg      string
		malleate func(icaAddress string)
		expErr   error
	}{
		{
			"interchain account successfully executes an arbitrary message type using the * (allow all message types) param",
			func(icaAddress string) {
				// Populate the gov keeper in advance with an active proposal
				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}

				proposalMsg, err := govv1.NewLegacyContent(testProposal, interchainAccountAddr)
				suite.Require().NoError(err)

				proposal, err := govv1.NewProposal([]sdk.Msg{proposalMsg}, govtypes.DefaultStartingProposalID, suite.chainA.GetContext().BlockTime(), suite.chainA.GetContext().BlockTime(), "test proposal", "title", "Description", sdk.AccAddress(interchainAccountAddr), false)
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().GovKeeper.SetProposal(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)

				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/cosmos.gov.v1beta1.MsgVote",
							"voter": "` + icaAddress + `",
							"proposal_id": 1,
							"option": 1
						}
					]
				}`)
				// this is the way cosmwasm encodes byte arrays by default
				// golang doesn't use this encoding by default, but it can still deserialize:
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",")

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{"*"})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes banktypes.MsgSend",
			func(icaAddress string) {
				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/cosmos.bank.v1beta1.MsgSend",
							"from_address": "` + icaAddress + `",
							"to_address": "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs",
							"amount": [{ "denom": "stake", "amount": "100" }]
						}
					]
				}`)
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",")

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*banktypes.MsgSend)(nil))})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes govtypes.MsgSubmitProposal",
			func(icaAddress string) {
				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/cosmos.gov.v1beta1.MsgSubmitProposal",
							"content": {
								"@type": "/cosmos.gov.v1beta1.TextProposal",
								"title": "IBC Gov Proposal",
								"description": "tokens for all!"
							},
							"initial_deposit": [{ "denom": "stake", "amount": "100000" }],
							"proposer": "` + icaAddress + `"
						}
					]
				}`)
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",")

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*govtypes.MsgSubmitProposal)(nil))})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes govtypes.MsgVote",
			func(icaAddress string) {
				// Populate the gov keeper in advance with an active proposal
				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}

				proposalMsg, err := govv1.NewLegacyContent(testProposal, interchainAccountAddr)
				suite.Require().NoError(err)

				proposal, err := govv1.NewProposal([]sdk.Msg{proposalMsg}, govtypes.DefaultStartingProposalID, suite.chainA.GetContext().BlockTime(), suite.chainA.GetContext().BlockTime(), "test proposal", "title", "description", sdk.AccAddress(interchainAccountAddr), false)
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().GovKeeper.SetProposal(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)

				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/cosmos.gov.v1beta1.MsgVote",
							"voter": "` + icaAddress + `",
							"proposal_id": 1,
							"option": 1
						}
					]
				}`)
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",")

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*govtypes.MsgVote)(nil))})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes govtypes.MsgSubmitProposal, govtypes.MsgDeposit, and then govtypes.MsgVote sequentially",
			func(icaAddress string) {
				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/cosmos.gov.v1beta1.MsgSubmitProposal",
							"content": {
								"@type": "/cosmos.gov.v1beta1.TextProposal",
								"title": "IBC Gov Proposal",
								"description": "tokens for all!"
							},
							"initial_deposit": [{ "denom": "stake", "amount": "100000" }],
							"proposer": "` + icaAddress + `"
						},
						{
							"@type": "/cosmos.gov.v1beta1.MsgDeposit",
							"proposal_id": 1,
							"depositor": "` + icaAddress + `",
							"amount": [{ "denom": "stake", "amount": "10000000" }]
						},
						{
							"@type": "/cosmos.gov.v1beta1.MsgVote",
							"voter": "` + icaAddress + `",
							"proposal_id": 1,
							"option": 1
						}
					]
				}`)
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",")

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*govtypes.MsgSubmitProposal)(nil)), sdk.MsgTypeURL((*govtypes.MsgDeposit)(nil)), sdk.MsgTypeURL((*govtypes.MsgVote)(nil))})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes transfertypes.MsgTransfer",
			func(icaAddress string) {
				transferPath := ibctesting.NewTransferPath(suite.chainB, suite.chainC)

				suite.coordinator.Setup(transferPath)

				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/ibc.applications.transfer.v1.MsgTransfer",
							"source_port": "transfer",
							"source_channel": "channel-1",
							"token": { "denom": "stake", "amount": "100" },
							"sender": "` + icaAddress + `",
							"receiver": "cosmos15ulrf36d4wdtrtqzkgaan9ylwuhs7k7qz753uk",
							"timeout_height": { "revision_number": 1, "revision_height": 100 },
							"timeout_timestamp": 0
						}
					]
				}`)
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",")

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*transfertypes.MsgTransfer)(nil))})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"unregistered sdk.Msg",
			func(icaAddress string) {
				msgBytes := []byte(`{"messages":[{}]}`)
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",")

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{"*"})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			icatypes.ErrUnknownDataType,
		},
		{
			"message type not allowed banktypes.MsgSend",
			func(icaAddress string) {
				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/cosmos.bank.v1beta1.MsgSend",
							"from_address": "` + icaAddress + `",
							"to_address": "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs",
							"amount": [{ "denom": "stake", "amount": "100" }]
						}
					]
				}`)
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",")

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*transfertypes.MsgTransfer)(nil))})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"unauthorised: signer address is not the interchain account associated with the controller portID",
			func(icaAddress string) {
				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/cosmos.bank.v1beta1.MsgSend",
							"from_address": "` + suite.chainB.SenderAccount.GetAddress().String() + `", // unexpected signer
							"to_address": "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs",
							"amount": [{ "denom": "stake", "amount": "100" }]
						}
					]
				}`)
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",")

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*banktypes.MsgSend)(nil))})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			icatypes.ErrUnknownDataType,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.msg, func() {
			suite.SetupTest() // reset

			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
			suite.Require().NoError(err)

			// Get the address of the interchain account stored in state during handshake step
			storedAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, portID)
			suite.Require().True(found)

			icaAddr, err := sdk.AccAddressFromBech32(storedAddr)
			suite.Require().NoError(err)

			// Check if account is created
			interchainAccount := suite.chainB.GetSimApp().AccountKeeper.GetAccount(suite.chainB.GetContext(), icaAddr)
			suite.Require().Equal(interchainAccount.GetAddress().String(), storedAddr)

			suite.fundICAWallet(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10000))))

			tc.malleate() // malleate mutates test data

			packet := channeltypes.NewPacket(
				packetData,
				suite.chainA.SenderAccount.GetSequence(),
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				clienttypes.NewHeight(1, 100),
				0,
			)

			txResponse, err := suite.chainB.GetSimApp().ICAHostKeeper.OnRecvPacket(suite.chainB.GetContext(), packet)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(txResponse)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(txResponse)
			}
		})
	}
}

func (suite *KeeperTestSuite) fundICAWallet(ctx sdk.Context, portID string, amount sdk.Coins) {
	interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(ctx, ibctesting.FirstConnectionID, portID)
	suite.Require().True(found)

	msgBankSend := &banktypes.MsgSend{
		FromAddress: suite.chainB.SenderAccount.GetAddress().String(),
		ToAddress:   interchainAccountAddr,
		Amount:      amount,
	}

	res, err := suite.chainB.SendMsgs(msgBankSend)
	suite.Require().NotEmpty(res)
	suite.Require().NoError(err)
}
