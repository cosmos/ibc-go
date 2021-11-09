package keeper_test

import (
	"fmt"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	transfertypes "github.com/cosmos/ibc-go/v2/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v2/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v2/testing"
)

func (suite *KeeperTestSuite) TestTrySendTx() {
	var (
		path       *ibctesting.Path
		packetData types.InterchainAccountPacketData
		chanCap    *capabilitytypes.Capability
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &banktypes.MsgSend{
					FromAddress: interchainAccountAddr,
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))),
				}

				data, err := types.SerializeCosmosTx(suite.chainB.GetSimApp().AppCodec(), []sdk.Msg{msg})
				suite.Require().NoError(err)

				packetData = types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}
			},
			true,
		},
		{
			"success with multiple sdk.Msg",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msgsBankSend := []sdk.Msg{
					&banktypes.MsgSend{
						FromAddress: interchainAccountAddr,
						ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
						Amount:      sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))),
					},
					&banktypes.MsgSend{
						FromAddress: interchainAccountAddr,
						ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
						Amount:      sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))),
					},
				}

				data, err := types.SerializeCosmosTx(suite.chainB.GetSimApp().AppCodec(), msgsBankSend)
				suite.Require().NoError(err)

				packetData = types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}
			},
			true,
		},
		{
			"data is nil",
			func() {
				packetData = types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: nil,
				}
			},
			false,
		},
		{
			"active channel not found",
			func() {
				path.EndpointA.ChannelConfig.PortID = "invalid-port-id"
			},
			false,
		},
		{
			"channel does not exist",
			func() {
				suite.chainA.GetSimApp().ICAKeeper.SetActiveChannelID(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, "channel-100")
			},
			false,
		},
		{
			"sendPacket fails - channel closed",
			func() {
				err := path.EndpointA.SetChannelClosed()
				suite.Require().NoError(err)
			},
			false,
		},
		{
			"invalid channel capability provided",
			func() {
				chanCap = nil
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := suite.SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			var ok bool
			chanCap, ok = suite.chainA.GetSimApp().ScopedICAMockKeeper.GetCapability(path.EndpointA.Chain.GetContext(), host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			suite.Require().True(ok)

			tc.malleate() // malleate mutates test data

			_, err = suite.chainA.GetSimApp().ICAKeeper.TrySendTx(suite.chainA.GetContext(), chanCap, path.EndpointA.ChannelConfig.PortID, packetData)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

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
			"interchain account successfully executes banktypes.MsgSend",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &banktypes.MsgSend{
					FromAddress: interchainAccountAddr,
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))),
				}

				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{msg})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			true,
		},
		{
			"interchain account successfully executes stakingtypes.MsgDelegate",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				validatorAddr := (sdk.ValAddress)(suite.chainB.Vals.Validators[0].Address)
				msg := &stakingtypes.MsgDelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin("stake", sdk.NewInt(5000)),
				}

				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{msg})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			true,
		},
		{
			"interchain account fails to execute stakingtypes.MsgDelegate - insufficient funds",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				validatorAddr := (sdk.ValAddress)(suite.chainB.Vals.Validators[0].Address)
				msg := &stakingtypes.MsgDelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin("stake", sdk.NewInt(50000)), // Increase the amount so it triggers insufficient funds
				}

				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{msg})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"interchain account successfully executes stakingtypes.MsgDelegate and stakingtypes.MsgUndelegate sequentially",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				validatorAddr := (sdk.ValAddress)(suite.chainB.Vals.Validators[0].Address)
				msgDelegate := &stakingtypes.MsgDelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin("stake", sdk.NewInt(5000)),
				}

				msgUndelegate := &stakingtypes.MsgUndelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin("stake", sdk.NewInt(5000)),
				}

				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{msgDelegate, msgUndelegate})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			true,
		},
		{
			"interchain account successfully executes govtypes.MsgSubmitProposal",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}

				any, err := codectypes.NewAnyWithValue(testProposal)
				suite.Require().NoError(err)

				msg := &govtypes.MsgSubmitProposal{
					Content:        any,
					InitialDeposit: sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(5000))),
					Proposer:       interchainAccountAddr,
				}

				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{msg})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			true,
		},
		{
			"interchain account successfully executes govtypes.MsgVote",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				// Populate the gov keeper in advance with an active proposal
				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}

				proposal, err := govtypes.NewProposal(testProposal, govtypes.DefaultStartingProposalID, time.Now(), time.Now().Add(time.Hour))
				suite.Require().NoError(err)

				suite.chainB.GetSimApp().GovKeeper.SetProposal(suite.chainB.GetContext(), proposal)
				suite.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(suite.chainB.GetContext(), proposal)

				msg := &govtypes.MsgVote{
					ProposalId: govtypes.DefaultStartingProposalID,
					Voter:      interchainAccountAddr,
					Option:     govtypes.OptionYes,
				}

				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{msg})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			true,
		},
		{
			"interchain account successfully executes disttypes.MsgFundCommunityPool",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &disttypes.MsgFundCommunityPool{
					Amount:    sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(5000))),
					Depositor: interchainAccountAddr,
				}

				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{msg})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			true,
		},
		{
			"interchain account successfully executes disttypes.MsgSetWithdrawAddress",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &disttypes.MsgSetWithdrawAddress{
					DelegatorAddress: interchainAccountAddr,
					WithdrawAddress:  suite.chainB.SenderAccount.GetAddress().String(),
				}

				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{msg})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			true,
		},
		{
			"interchain account successfully executes transfertypes.MsgTransfer",
			func() {
				transferPath := ibctesting.NewPath(suite.chainB, suite.chainC)
				transferPath.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
				transferPath.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort

				suite.coordinator.Setup(transferPath)

				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &transfertypes.MsgTransfer{
					SourcePort:       transferPath.EndpointA.ChannelConfig.PortID,
					SourceChannel:    transferPath.EndpointA.ChannelID,
					Token:            sdk.NewCoin("stake", sdk.NewInt(100)),
					Sender:           interchainAccountAddr,
					Receiver:         suite.chainA.SenderAccount.GetAddress().String(),
					TimeoutHeight:    clienttypes.NewHeight(0, 100),
					TimeoutTimestamp: uint64(0),
				}

				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{msg})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			true,
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

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"invalid packet type - UNSPECIFIED",
			func() {
				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{&banktypes.MsgSend{}})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.UNSPECIFIED,
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

				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{&banktypes.MsgSend{}})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"unauthorised: unexpected signer address",
			func() {
				msg := &banktypes.MsgSend{
					FromAddress: suite.chainB.SenderAccount.GetAddress().String(), // unexpected signer
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100))),
				}

				data, err := types.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []sdk.Msg{msg})
				suite.Require().NoError(err)

				icaPacketData := types.InterchainAccountPacketData{
					Type: types.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := suite.SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			suite.fundICAWallet(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(10000))))

			tc.malleate() // malleate mutates test data

			packet := channeltypes.NewPacket(
				packetData,
				suite.chainA.SenderAccount.GetSequence(),
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			err = suite.chainB.GetSimApp().ICAKeeper.OnRecvPacket(suite.chainB.GetContext(), packet)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestOnTimeoutPacket() {
	var (
		path *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := suite.SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			packet := channeltypes.NewPacket(
				[]byte{},
				1,
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			err = suite.chainA.GetSimApp().ICAKeeper.OnTimeoutPacket(suite.chainA.GetContext(), packet)

			activeChannelID, found := suite.chainA.GetSimApp().ICAKeeper.GetActiveChannelID(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Empty(activeChannelID)
				suite.Require().False(found)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) fundICAWallet(ctx sdk.Context, portID string, amount sdk.Coins) {
	interchainAccountAddr, found := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(ctx, portID)
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
