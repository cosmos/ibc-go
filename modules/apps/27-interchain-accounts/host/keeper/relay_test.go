package keeper_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *KeeperTestSuite) TestOnRecvPacket() {
	testedOrderings := []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED}
	testedEncodings := []string{icatypes.EncodingProtobuf, icatypes.EncodingProto3JSON}

	var (
		path       *ibctesting.Path
		packetData []byte
	)

	testCases := []struct {
		msg      string
		malleate func(encoding string)
		expErr   error
	}{
		{
			"interchain account successfully executes an arbitrary message type using the * (allow all message types) param",
			func(encoding string) {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				proposal, err := govtypesv1.NewProposal([]sdk.Msg{getTestProposalMessage()}, govtypesv1.DefaultStartingProposalID, time.Now(), time.Now().Add(time.Hour), "test proposal", "title", "Description", sdk.AccAddress(interchainAccountAddr), false)
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().GovKeeper.SetProposal(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)

				msg := &govtypesv1.MsgVote{
					ProposalId: govtypesv1.DefaultStartingProposalID,
					Voter:      interchainAccountAddr,
					Option:     govtypesv1.OptionYes,
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{"*"})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes banktypes.MsgSend",
			func(encoding string) {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &banktypes.MsgSend{
					FromAddress: interchainAccountAddr,
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(ibctesting.TestCoin),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes stakingtypes.MsgDelegate",
			func(encoding string) {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				validatorAddr := (sdk.ValAddress)(suite.chainB.Vals.Validators[0].Address)
				msg := &stakingtypes.MsgDelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes stakingtypes.MsgDelegate and stakingtypes.MsgUndelegate sequentially",
			func(encoding string) {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				validatorAddr := (sdk.ValAddress)(suite.chainB.Vals.Validators[0].Address)
				msgDelegate := &stakingtypes.MsgDelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
				}

				msgUndelegate := &stakingtypes.MsgUndelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msgDelegate, msgUndelegate}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msgDelegate), sdk.MsgTypeURL(msgUndelegate)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes govtypesv1.MsgSubmitProposal",
			func(encoding string) {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg, err := govtypesv1.NewMsgSubmitProposal([]sdk.Msg{}, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000))), interchainAccountAddr, "metadata", "title", "summary", false)
				suite.Require().NoError(err)

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes govtypesv1.MsgVote",
			func(encoding string) {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				proposal, err := govtypesv1.NewProposal([]sdk.Msg{getTestProposalMessage()}, govtypesv1.DefaultStartingProposalID, time.Now(), time.Now().Add(time.Hour), "test proposal", "title", "Description", sdk.AccAddress(interchainAccountAddr), false)
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().GovKeeper.SetProposal(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)

				msg := &govtypesv1.MsgVote{
					ProposalId: govtypesv1.DefaultStartingProposalID,
					Voter:      interchainAccountAddr,
					Option:     govtypesv1.OptionYes,
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes disttypes.MsgFundCommunityPool",
			func(encoding string) {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &disttypes.MsgFundCommunityPool{
					Amount:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000))),
					Depositor: interchainAccountAddr,
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes icahosttypes.MsgModuleQuerySafe",
			func(encoding string) {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				balanceQuery := banktypes.NewQueryBalanceRequest(suite.chainB.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
				queryBz, err := balanceQuery.Marshal()
				suite.Require().NoError(err)

				msg := types.NewMsgModuleQuerySafe(interchainAccountAddr, []types.QueryRequest{
					{
						Path: "/cosmos.bank.v1beta1.Query/Balance",
						Data: queryBz,
					},
				})

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes disttypes.MsgSetWithdrawAddress",
			func(encoding string) {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &disttypes.MsgSetWithdrawAddress{
					DelegatorAddress: interchainAccountAddr,
					WithdrawAddress:  suite.chainB.SenderAccount.GetAddress().String(),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes transfertypes.MsgTransfer",
			func(encoding string) {
				transferPath := ibctesting.NewTransferPath(suite.chainB, suite.chainC)

				transferPath.Setup()

				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := transfertypes.NewMsgTransfer(
					transferPath.EndpointA.ChannelConfig.PortID,
					transferPath.EndpointA.ChannelID,
					ibctesting.TestCoin,
					interchainAccountAddr,
					suite.chainA.SenderAccount.GetAddress().String(),
					suite.chainB.GetTimeoutHeight(),
					0,
					"",
				)

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"Msg fails its ValidateBasic: MsgTransfer has an empty receiver",
			func(encoding string) {
				transferPath := ibctesting.NewTransferPath(suite.chainB, suite.chainC)
				transferPath.Setup()

				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := transfertypes.NewMsgTransfer(
					transferPath.EndpointA.ChannelConfig.PortID,
					transferPath.EndpointA.ChannelID,
					ibctesting.TestCoin,
					interchainAccountAddr,
					"",
					suite.chainB.GetTimeoutHeight(),
					0,
					"",
				)

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"unregistered sdk.Msg",
			func(encoding string) {
				msg := &banktypes.MsgSendResponse{}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{"/" + proto.MessageName(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"cannot unmarshal interchain account packet data",
			func(encoding string) {
				packetData = []byte{}
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"cannot deserialize interchain account packet data messages",
			func(encoding string) {
				data := []byte("invalid packet data")

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"invalid packet type - UNSPECIFIED",
			func(encoding string) {
				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{&banktypes.MsgSend{}}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.UNSPECIFIED,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			icatypes.ErrUnknownDataType,
		},
		{
			"unauthorised: interchain account not found for controller port ID",
			func(encoding string) {
				path.EndpointA.ChannelConfig.PortID = "invalid-port-id"

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{&banktypes.MsgSend{}}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			icatypes.ErrInterchainAccountNotFound,
		},
		{
			"unauthorised: message type not allowed", // NOTE: do not update params to explicitly force the error
			func(encoding string) {
				msg := &banktypes.MsgSend{
					FromAddress: suite.chainB.SenderAccount.GetAddress().String(),
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(ibctesting.TestCoin),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"unauthorised: signer address is not the interchain account associated with the controller portID",
			func(encoding string) {
				msg := &banktypes.MsgSend{
					FromAddress: suite.chainB.SenderAccount.GetAddress().String(), // unexpected signer
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(ibctesting.TestCoin),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
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

	for _, ordering := range testedOrderings {
		for _, encoding := range testedEncodings {
			for _, tc := range testCases {
				tc := tc

				suite.Run(tc.msg, func() {
					suite.SetupTest() // reset

					path = NewICAPath(suite.chainA, suite.chainB, encoding, ordering)
					path.SetupConnections()

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

					if tc.expErr == nil {
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
				proposal, err := govtypesv1.NewProposal([]sdk.Msg{getTestProposalMessage()}, govtypesv1.DefaultStartingProposalID, suite.chainA.GetContext().BlockTime(), suite.chainA.GetContext().BlockTime(), "test proposal", "title", "Description", sdk.AccAddress(interchainAccountAddr), false)
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().GovKeeper.SetProposal(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)

				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/cosmos.gov.v1.MsgVote",
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
			"interchain account successfully executes govtypesv1.MsgSubmitProposal",
			func(icaAddress string) {
				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/cosmos.gov.v1.MsgSubmitProposal",
							"messages": [],
							"metadata": "ipfs://CID",
 							"title": "IBC Gov Proposal",
							"summary": "tokens for all!",
							"expedited": false,
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

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*govtypesv1.MsgSubmitProposal)(nil))})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes govtypesv1.MsgVote",
			func(icaAddress string) {
				proposal, err := govtypesv1.NewProposal([]sdk.Msg{getTestProposalMessage()}, govtypesv1.DefaultStartingProposalID, suite.chainA.GetContext().BlockTime(), suite.chainA.GetContext().BlockTime(), "test proposal", "title", "Description", sdk.AccAddress(interchainAccountAddr), false)
				suite.Require().NoError(err)

				err = suite.chainB.GetSimApp().GovKeeper.SetProposal(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)
				err = suite.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(suite.chainB.GetContext(), proposal)
				suite.Require().NoError(err)

				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/cosmos.gov.v1.MsgVote",
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

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*govtypesv1.MsgVote)(nil))})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes govtypesv1.MsgSubmitProposal, govtypesv1.MsgDeposit, and then govtypesv1.MsgVote sequentially",
			func(icaAddress string) {
				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/cosmos.gov.v1.MsgSubmitProposal",
							"messages": [],
							"metadata": "ipfs://CID",
 							"title": "IBC Gov Proposal",
							"summary": "tokens for all!",
							"expedited": false,
							"initial_deposit": [{ "denom": "stake", "amount": "100000" }],
							"proposer": "` + icaAddress + `"
						},
						{
							"@type": "/cosmos.gov.v1.MsgDeposit",
							"proposal_id": 1,
							"depositor": "` + icaAddress + `",
							"amount": [{ "denom": "stake", "amount": "10000000" }]
						},
						{
							"@type": "/cosmos.gov.v1.MsgVote",
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

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*govtypesv1.MsgSubmitProposal)(nil)), sdk.MsgTypeURL((*govtypesv1.MsgDeposit)(nil)), sdk.MsgTypeURL((*govtypesv1.MsgVote)(nil))})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes transfertypes.MsgTransfer",
			func(icaAddress string) {
				transferPath := ibctesting.NewTransferPath(suite.chainB, suite.chainC)

				transferPath.Setup()

				msgBytes := []byte(`{
					"messages": [
						{
							"@type": "/ibc.applications.transfer.v1.MsgTransfer",
							"source_port": "transfer",
							"source_channel": "` + transferPath.EndpointA.ChannelID + `",
							"token": { "denom": "stake", "amount": "100" },
							"sender": "` + icaAddress + `",
							"receiver": "cosmos15ulrf36d4wdtrtqzkgaan9ylwuhs7k7qz753uk",
							"timeout_height": { "revision_number": 1, "revision_height": 100 },
							"timeout_timestamp": 0,
							"memo": ""
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
			ibcerrors.ErrInvalidType,
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
			ibcerrors.ErrInvalidType,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			tc := tc

			suite.Run(tc.msg, func() {
				suite.SetupTest() // reset

				path = NewICAPath(suite.chainA, suite.chainB, icatypes.EncodingProto3JSON, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				suite.Require().NoError(err)

				portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
				suite.Require().NoError(err)

				// Get the address of the interchain account stored in state during handshake step
				icaAddress, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, portID)
				suite.Require().True(found)

				suite.fundICAWallet(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000000))))

				tc.malleate(icaAddress) // malleate mutates test data

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

				if tc.expErr == nil {
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

func getTestProposalMessage() sdk.Msg {
	_, _, addr := testdata.KeyTestPubAddr()
	return banktypes.NewMsgSend(authtypes.NewModuleAddress("gov"), addr, sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(1000))))
}
