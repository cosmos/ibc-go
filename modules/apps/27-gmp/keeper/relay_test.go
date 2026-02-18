package keeper_test

import (
	"testing"

	"github.com/cosmos/gogoproto/proto"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const testSalt = "test-salt"

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
}

func (s *KeeperTestSuite) TestAuthenticateTx() {
	var (
		gmpKeeper *keeper.Keeper
		account   sdk.AccountI
		msgs      []sdk.Msg
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: single message",
			func() {
				msgs = []sdk.Msg{s.newMsgSend(account.GetAddress(), s.chainB.SenderAccount.GetAddress())}
			},
			nil,
		},
		{
			"success: multiple messages",
			func() {
				msgs = []sdk.Msg{
					s.newMsgSend(account.GetAddress(), s.chainB.SenderAccount.GetAddress()),
					s.newMsgSend(account.GetAddress(), s.chainB.SenderAccount.GetAddress()),
				}
			},
			nil,
		},
		{
			"failure: empty messages",
			func() {
				msgs = []sdk.Msg{}
			},
			types.ErrInvalidPayload,
		},
		{
			"failure: wrong signer",
			func() {
				msgs = []sdk.Msg{s.newMsgSend(s.chainB.SenderAccount.GetAddress(), s.chainA.SenderAccount.GetAddress())}
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: one wrong signer in multiple messages",
			func() {
				msgs = []sdk.Msg{
					s.newMsgSend(account.GetAddress(), s.chainB.SenderAccount.GetAddress()),
					s.newMsgSend(s.chainB.SenderAccount.GetAddress(), s.chainA.SenderAccount.GetAddress()),
				}
			},
			ibcerrors.ErrUnauthorized,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			gmpKeeper = s.chainA.GetSimApp().GMPKeeper
			account = s.chainA.SenderAccount

			tc.malleate()

			err := gmpKeeper.AuthenticateTx(s.chainA.GetContext(), account, msgs)
			expPass := tc.expErr == nil
			if expPass {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestOnRecvPacket() {
	var (
		gmpKeeper      *keeper.Keeper
		packetData     *types.GMPPacketData
		gmpAccountAddr sdk.AccAddress
		sender         string
		recipient      sdk.AccAddress
		destClient     string
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success: bank transfer",
			func() {
				s.fundAccount(gmpAccountAddr, sdk.NewCoins(ibctesting.TestCoin))
				packetData.Payload = s.serializeMsgs(s.newMsgSend(gmpAccountAddr, recipient))
			},
			nil,
		},
		{
			"success: multiple messages",
			func() {
				amount := sdk.NewCoin(ibctesting.TestCoin.Denom, sdkmath.NewInt(500))
				s.fundAccount(gmpAccountAddr, sdk.NewCoins(sdk.NewCoin(ibctesting.TestCoin.Denom, sdkmath.NewInt(2000))))
				packetData.Payload = s.serializeMsgs(
					s.newMsgSendWithAmount(gmpAccountAddr, recipient, amount),
					s.newMsgSendWithAmount(gmpAccountAddr, recipient, amount),
				)
			},
			nil,
		},
		{
			"failure: unauthorized signer",
			func() {
				senderAddr, _ := sdk.AccAddressFromBech32(sender)
				packetData.Payload = s.serializeMsgs(s.newMsgSend(senderAddr, recipient))
			},
			ibcerrors.ErrUnauthorized,
		},
		{
			"failure: invalid payload",
			func() {
				packetData.Payload = []byte("invalid")
			},
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: empty payload",
			func() {
				packetData.Payload = []byte{}
			},
			types.ErrInvalidPayload,
		},
		{
			"failure: msg ValidateBasic error - invalid to address",
			func() {
				invalidMsg := &banktypes.MsgSend{
					FromAddress: gmpAccountAddr.String(),
					ToAddress:   "invalid",
					Amount:      sdk.NewCoins(ibctesting.TestCoin),
				}
				packetData.Payload = s.serializeMsgs(invalidMsg)
			},
			sdkerrors.ErrInvalidAddress,
		},
		{
			"failure: getOrCreateICS27Account error - invalid dest client",
			func() {
				destClient = "x" // too short, fails ClientIdentifierValidator
			},
			host.ErrInvalidID,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			gmpKeeper = s.chainA.GetSimApp().GMPKeeper
			sender = s.chainB.SenderAccount.GetAddress().String()
			recipient = s.chainA.SenderAccount.GetAddress()
			destClient = ibctesting.FirstClientID

			accountID := types.NewAccountIdentifier(ibctesting.FirstClientID, sender, []byte(testSalt))
			addr, err := types.BuildAddressPredictable(&accountID)
			s.Require().NoError(err)
			gmpAccountAddr = addr

			data := types.NewGMPPacketData(sender, "", []byte(testSalt), nil, "")
			packetData = &data

			tc.malleate()

			result, err := gmpKeeper.OnRecvPacket(
				s.chainA.GetContext(),
				packetData,
				destClient,
			)

			expPass := tc.expErr == nil
			if expPass {
				s.Require().NoError(err)
				s.Require().NotEmpty(result)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
				s.Require().Nil(result)
			}
		})
	}
}

func (s *KeeperTestSuite) fundAccount(addr sdk.AccAddress, coins sdk.Coins) {
	err := s.chainA.GetSimApp().BankKeeper.SendCoins(
		s.chainA.GetContext(),
		s.chainA.SenderAccount.GetAddress(),
		addr,
		coins,
	)
	s.Require().NoError(err)
}

func (s *KeeperTestSuite) newMsgSend(from, to sdk.AccAddress) *banktypes.MsgSend {
	return s.newMsgSendWithAmount(from, to, ibctesting.TestCoin)
}

func (s *KeeperTestSuite) newMsgSendWithAmount(from, to sdk.AccAddress, amount sdk.Coin) *banktypes.MsgSend {
	return &banktypes.MsgSend{
		FromAddress: from.String(),
		ToAddress:   to.String(),
		Amount:      sdk.NewCoins(amount),
	}
}

func (s *KeeperTestSuite) serializeMsgs(msgs ...proto.Message) []byte {
	payload, err := types.SerializeCosmosTx(s.chainA.GetSimApp().AppCodec(), msgs)
	s.Require().NoError(err)
	return payload
}
