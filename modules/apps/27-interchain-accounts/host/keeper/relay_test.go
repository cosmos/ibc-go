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

	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestOnRecvPacket() {
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
				interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				proposal, err := govtypesv1.NewProposal([]sdk.Msg{getTestProposalMessage()}, govtypesv1.DefaultStartingProposalID, time.Now(), time.Now().Add(time.Hour), "test proposal", "title", "Description", sdk.AccAddress(interchainAccountAddr), false)
				s.Require().NoError(err)

				err = s.chainB.GetSimApp().GovKeeper.SetProposal(s.chainB.GetContext(), proposal)
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(s.chainB.GetContext(), proposal)
				s.Require().NoError(err)

				msg := &govtypesv1.MsgVote{
					ProposalId: govtypesv1.DefaultStartingProposalID,
					Voter:      interchainAccountAddr,
					Option:     govtypesv1.OptionYes,
				}

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{"*"})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes banktypes.MsgSend",
			func(encoding string) {
				interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				msg := &banktypes.MsgSend{
					FromAddress: interchainAccountAddr,
					ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(ibctesting.TestCoin),
				}

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes stakingtypes.MsgDelegate",
			func(encoding string) {
				interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				validatorAddr := (sdk.ValAddress)(s.chainB.Vals.Validators[0].Address)
				msg := &stakingtypes.MsgDelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
				}

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes stakingtypes.MsgDelegate and stakingtypes.MsgUndelegate sequentially",
			func(encoding string) {
				interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				validatorAddr := (sdk.ValAddress)(s.chainB.Vals.Validators[0].Address)
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

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msgDelegate, msgUndelegate}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msgDelegate), sdk.MsgTypeURL(msgUndelegate)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes govtypesv1.MsgSubmitProposal",
			func(encoding string) {
				interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				msg, err := govtypesv1.NewMsgSubmitProposal([]sdk.Msg{}, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000))), interchainAccountAddr, "metadata", "title", "summary", false)
				s.Require().NoError(err)

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes govtypesv1.MsgVote",
			func(encoding string) {
				interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				proposal, err := govtypesv1.NewProposal([]sdk.Msg{getTestProposalMessage()}, govtypesv1.DefaultStartingProposalID, time.Now(), time.Now().Add(time.Hour), "test proposal", "title", "Description", sdk.AccAddress(interchainAccountAddr), false)
				s.Require().NoError(err)

				err = s.chainB.GetSimApp().GovKeeper.SetProposal(s.chainB.GetContext(), proposal)
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(s.chainB.GetContext(), proposal)
				s.Require().NoError(err)

				msg := &govtypesv1.MsgVote{
					ProposalId: govtypesv1.DefaultStartingProposalID,
					Voter:      interchainAccountAddr,
					Option:     govtypesv1.OptionYes,
				}

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes disttypes.MsgFundCommunityPool",
			func(encoding string) {
				interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				msg := &disttypes.MsgFundCommunityPool{
					Amount:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000))),
					Depositor: interchainAccountAddr,
				}

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes icahosttypes.MsgModuleQuerySafe",
			func(encoding string) {
				interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				balanceQuery := banktypes.NewQueryBalanceRequest(s.chainB.SenderAccount.GetAddress(), sdk.DefaultBondDenom)
				queryBz, err := balanceQuery.Marshal()
				s.Require().NoError(err)

				msg := types.NewMsgModuleQuerySafe(interchainAccountAddr, []types.QueryRequest{
					{
						Path: "/cosmos.bank.v1beta1.Query/Balance",
						Data: queryBz,
					},
				})

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes disttypes.MsgSetWithdrawAddress",
			func(encoding string) {
				interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				msg := &disttypes.MsgSetWithdrawAddress{
					DelegatorAddress: interchainAccountAddr,
					WithdrawAddress:  s.chainB.SenderAccount.GetAddress().String(),
				}

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes transfertypes.MsgTransfer",
			func(encoding string) {
				transferPath := ibctesting.NewTransferPath(s.chainB, s.chainC)

				transferPath.Setup()

				interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				msg := transfertypes.NewMsgTransfer(
					transferPath.EndpointA.ChannelConfig.PortID,
					transferPath.EndpointA.ChannelID,
					ibctesting.TestCoin,
					interchainAccountAddr,
					s.chainA.SenderAccount.GetAddress().String(),
					s.chainB.GetTimeoutHeight(),
					0,
					"",
				)

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"Msg fails its ValidateBasic: MsgTransfer has an empty receiver",
			func(encoding string) {
				transferPath := ibctesting.NewTransferPath(s.chainB, s.chainC)
				transferPath.Setup()

				interchainAccountAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				s.Require().True(found)

				msg := transfertypes.NewMsgTransfer(
					transferPath.EndpointA.ChannelConfig.PortID,
					transferPath.EndpointA.ChannelID,
					ibctesting.TestCoin,
					interchainAccountAddr,
					"",
					s.chainB.GetTimeoutHeight(),
					0,
					"",
				)

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"unregistered sdk.Msg",
			func(encoding string) {
				msg := &banktypes.MsgSendResponse{}

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{"/" + proto.MessageName(msg)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
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
				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{&banktypes.MsgSend{}}, encoding)
				s.Require().NoError(err)

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
				path.EndpointA.ChannelConfig.PortID = "invalid-port-id" //nolint:goconst

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{&banktypes.MsgSend{}}, encoding)
				s.Require().NoError(err)

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
					FromAddress: s.chainB.SenderAccount.GetAddress().String(),
					ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(ibctesting.TestCoin),
				}

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

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
					FromAddress: s.chainB.SenderAccount.GetAddress().String(), // unexpected signer
					ToAddress:   s.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(ibctesting.TestCoin),
				}

				data, err := icatypes.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, encoding)
				s.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, ordering := range testedOrderings {
		for _, encoding := range testedEncodings {
			for _, tc := range testCases {
				s.Run(tc.msg, func() {
					s.SetupTest() // reset

					path = NewICAPath(s.chainA, s.chainB, encoding, ordering)
					path.SetupConnections()

					err := SetupICAPath(path, TestOwnerAddress)
					s.Require().NoError(err)

					portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
					s.Require().NoError(err)

					// Get the address of the interchain account stored in state during handshake step
					storedAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, portID)
					s.Require().True(found)

					icaAddr, err := sdk.AccAddressFromBech32(storedAddr)
					s.Require().NoError(err)

					// Check if account is created
					interchainAccount := s.chainB.GetSimApp().AccountKeeper.GetAccount(s.chainB.GetContext(), icaAddr)
					s.Require().Equal(interchainAccount.GetAddress().String(), storedAddr)

					s.fundICAWallet(s.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1000000))))

					tc.malleate(encoding) // malleate mutates test data

					packet := channeltypes.NewPacket(
						packetData,
						s.chainA.SenderAccount.GetSequence(),
						path.EndpointA.ChannelConfig.PortID,
						path.EndpointA.ChannelID,
						path.EndpointB.ChannelConfig.PortID,
						path.EndpointB.ChannelID,
						s.chainB.GetTimeoutHeight(),
						0,
					)

					txResponse, err := s.chainB.GetSimApp().ICAHostKeeper.OnRecvPacket(s.chainB.GetContext(), packet)

					if tc.expErr == nil {
						s.Require().NoError(err)
						s.Require().NotNil(txResponse)
					} else {
						s.Require().ErrorIs(err, tc.expErr)
						s.Require().Nil(txResponse)
					}
				})
			}
		}
	}
}

func (s *KeeperTestSuite) TestJSONOnRecvPacket() {
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
				proposal, err := govtypesv1.NewProposal([]sdk.Msg{getTestProposalMessage()}, govtypesv1.DefaultStartingProposalID, s.chainA.GetContext().BlockTime(), s.chainA.GetContext().BlockTime(), "test proposal", "title", "Description", sdk.AccAddress(interchainAccountAddr), false)
				s.Require().NoError(err)

				err = s.chainB.GetSimApp().GovKeeper.SetProposal(s.chainB.GetContext(), proposal)
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(s.chainB.GetContext(), proposal)
				s.Require().NoError(err)

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
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",") //nolint:staticcheck

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{"*"})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
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
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",") //nolint:staticcheck

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*banktypes.MsgSend)(nil))})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
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
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",") //nolint:staticcheck

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*govtypesv1.MsgSubmitProposal)(nil))})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes govtypesv1.MsgVote",
			func(icaAddress string) {
				proposal, err := govtypesv1.NewProposal([]sdk.Msg{getTestProposalMessage()}, govtypesv1.DefaultStartingProposalID, s.chainA.GetContext().BlockTime(), s.chainA.GetContext().BlockTime(), "test proposal", "title", "Description", sdk.AccAddress(interchainAccountAddr), false)
				s.Require().NoError(err)

				err = s.chainB.GetSimApp().GovKeeper.SetProposal(s.chainB.GetContext(), proposal)
				s.Require().NoError(err)
				err = s.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(s.chainB.GetContext(), proposal)
				s.Require().NoError(err)

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
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",") //nolint:staticcheck

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*govtypesv1.MsgVote)(nil))})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
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
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",") //nolint:staticcheck

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*govtypesv1.MsgSubmitProposal)(nil)), sdk.MsgTypeURL((*govtypesv1.MsgDeposit)(nil)), sdk.MsgTypeURL((*govtypesv1.MsgVote)(nil))})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"interchain account successfully executes transfertypes.MsgTransfer",
			func(icaAddress string) {
				transferPath := ibctesting.NewTransferPath(s.chainB, s.chainC)

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
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",") //nolint:staticcheck

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*transfertypes.MsgTransfer)(nil))})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			nil,
		},
		{
			"unregistered sdk.Msg",
			func(icaAddress string) {
				msgBytes := []byte(`{"messages":[{}]}`)
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",") //nolint:staticcheck

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{"*"})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
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
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",") //nolint:staticcheck

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*transfertypes.MsgTransfer)(nil))})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
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
							"from_address": "` + s.chainB.SenderAccount.GetAddress().String() + `", // unexpected signer
							"to_address": "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs",
							"amount": [{ "denom": "stake", "amount": "100" }]
						}
					]
				}`)
				byteArrayString := strings.Join(strings.Fields(fmt.Sprint(msgBytes)), ",") //nolint:staticcheck

				packetData = []byte(`{
					"type": 1,
					"data":` + byteArrayString + `
				}`)

				params := types.NewParams(true, []string{sdk.MsgTypeURL((*banktypes.MsgSend)(nil))})
				s.chainB.GetSimApp().ICAHostKeeper.SetParams(s.chainB.GetContext(), params)
			},
			ibcerrors.ErrInvalidType,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.msg, func() {
				s.SetupTest() // reset

				path = NewICAPath(s.chainA, s.chainB, icatypes.EncodingProto3JSON, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				s.Require().NoError(err)

				portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
				s.Require().NoError(err)

				// Get the address of the interchain account stored in state during handshake step
				icaAddress, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, portID)
				s.Require().True(found)

				s.fundICAWallet(s.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000000))))

				tc.malleate(icaAddress) // malleate mutates test data

				packet := channeltypes.NewPacket(
					packetData,
					s.chainA.SenderAccount.GetSequence(),
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					path.EndpointB.ChannelConfig.PortID,
					path.EndpointB.ChannelID,
					s.chainB.GetTimeoutHeight(),
					0,
				)

				txResponse, err := s.chainB.GetSimApp().ICAHostKeeper.OnRecvPacket(s.chainB.GetContext(), packet)

				if tc.expErr == nil {
					s.Require().NoError(err)
					s.Require().NotNil(txResponse)
				} else {
					s.Require().ErrorIs(err, tc.expErr)
					s.Require().Nil(txResponse)
				}
			})
		}
	}
}

func (s *KeeperTestSuite) fundICAWallet(ctx sdk.Context, portID string, amount sdk.Coins) {
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

func getTestProposalMessage() sdk.Msg {
	_, _, addr := testdata.KeyTestPubAddr()
	return banktypes.NewMsgSend(authtypes.NewModuleAddress("gov"), addr, sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(1000))))
}
