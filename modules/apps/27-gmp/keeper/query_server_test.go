package keeper_test

import (
	"encoding/hex"
	"errors"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *KeeperTestSuite) TestQueryAccountAddress() {
	var req *types.QueryAccountAddressRequest

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
			"success: empty salt",
			func() {
				req.Salt = ""
			},
			nil,
		},
		{
			"failure: invalid salt hex",
			func() {
				req.Salt = "not-hex"
			},
			hex.InvalidByteError('n'),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			req = &types.QueryAccountAddressRequest{
				ClientId: ibctesting.FirstClientID,
				Sender:   s.chainA.SenderAccount.GetAddress().String(),
				Salt:     hex.EncodeToString([]byte(testSalt)),
			}

			tc.malleate()

			resp, err := s.chainA.GetSimApp().GMPKeeper.AccountAddress(s.chainA.GetContext(), req)

			expPass := tc.expErr == nil
			if expPass {
				s.Require().NoError(err)
				s.Require().NotEmpty(resp.AccountAddress)

				_, err := sdk.AccAddressFromBech32(resp.AccountAddress)
				s.Require().NoError(err)
			} else {
				s.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}

func (s *KeeperTestSuite) TestQueryAccountIdentifier() {
	var (
		req            *types.QueryAccountIdentifierRequest
		gmpAccountAddr string
		expAccountID   *types.AccountIdentifier
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success",
			func() {
				sender := s.chainB.SenderAccount.GetAddress().String()
				expAccountID = &types.AccountIdentifier{
					ClientId: ibctesting.FirstClientID,
					Sender:   sender,
					Salt:     []byte(testSalt),
				}
				s.createGMPAccount(gmpAccountAddr)
			},
			nil,
		},
		{
			"failure: invalid address",
			func() {
				req.AccountAddress = "invalid"
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: account not found",
			func() {},
			collections.ErrNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			sender := s.chainB.SenderAccount.GetAddress().String()
			accountID := types.NewAccountIdentifier(ibctesting.FirstClientID, sender, []byte(testSalt))
			addr, err := types.BuildAddressPredictable(&accountID)
			s.Require().NoError(err)
			gmpAccountAddr = addr.String()

			req = &types.QueryAccountIdentifierRequest{
				AccountAddress: gmpAccountAddr,
			}

			tc.malleate()

			resp, err := s.chainA.GetSimApp().GMPKeeper.AccountIdentifier(s.chainA.GetContext(), req)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().Equal(expAccountID.ClientId, resp.AccountId.ClientId)
				s.Require().Equal(expAccountID.Sender, resp.AccountId.Sender)
				s.Require().Equal(expAccountID.Salt, resp.AccountId.Salt)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetAccount() {
	testCases := []struct {
		name     string
		malleate func(addr sdk.AccAddress)
		expErr   error
	}{
		{
			"success",
			func(addr sdk.AccAddress) {
				s.createGMPAccount(addr.String())
			},
			nil,
		},
		{
			"failure: account not found",
			func(addr sdk.AccAddress) {},
			collections.ErrNotFound,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			sender := s.chainB.SenderAccount.GetAddress().String()
			accountID := types.NewAccountIdentifier(ibctesting.FirstClientID, sender, []byte(testSalt))
			addr, err := types.BuildAddressPredictable(&accountID)
			s.Require().NoError(err)

			tc.malleate(addr)

			account, err := s.chainA.GetSimApp().GMPKeeper.GetAccount(s.chainA.GetContext(), addr)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotNil(account)
				s.Require().Equal(addr.String(), account.Address)
			} else {
				s.Require().ErrorIs(err, tc.expErr)
				s.Require().Nil(account)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetOrComputeICS27Address() {
	testCases := []struct {
		name      string
		accountID *types.AccountIdentifier
		expErr    error
	}{
		{
			"success: existing account",
			&types.AccountIdentifier{
				ClientId: ibctesting.FirstClientID,
				Sender:   "", // will be set in test
				Salt:     []byte(testSalt),
			},
			nil,
		},
		{
			"success: computes new address",
			&types.AccountIdentifier{
				ClientId: ibctesting.FirstClientID,
				Sender:   "", // will be set in test
				Salt:     []byte("new-salt"),
			},
			nil,
		},
		{
			"failure: invalid client ID",
			&types.AccountIdentifier{
				ClientId: "invalid",
				Sender:   "", // will be set in test
				Salt:     []byte(testSalt),
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: empty sender",
			&types.AccountIdentifier{
				ClientId: ibctesting.FirstClientID,
				Sender:   "",
				Salt:     []byte(testSalt),
			},
			ibcerrors.ErrInvalidAddress,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			sender := s.chainB.SenderAccount.GetAddress().String()

			// Set sender if not testing empty sender case
			if tc.accountID.Sender == "" && !errors.Is(tc.expErr, ibcerrors.ErrInvalidAddress) {
				tc.accountID.Sender = sender
			}

			// Create existing account for first test case
			if tc.name == "success: existing account" {
				accountID := types.NewAccountIdentifier(ibctesting.FirstClientID, sender, []byte(testSalt))
				addr, err := types.BuildAddressPredictable(&accountID)
				s.Require().NoError(err)
				s.createGMPAccount(addr.String())
			}

			addr, err := s.chainA.GetSimApp().GMPKeeper.GetOrComputeICS27Address(
				s.chainA.GetContext(),
				tc.accountID,
			)

			if tc.expErr == nil {
				s.Require().NoError(err)
				s.Require().NotEmpty(addr)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *KeeperTestSuite) createGMPAccount(gmpAccountAddr string) {
	sender := s.chainB.SenderAccount.GetAddress().String()
	recipient := s.chainA.SenderAccount.GetAddress()

	gmpAddr, _ := sdk.AccAddressFromBech32(gmpAccountAddr)
	s.fundAccount(gmpAddr, sdk.NewCoins(ibctesting.TestCoin))

	data := types.NewGMPPacketData(sender, "", []byte(testSalt), nil, "")
	data.Payload = s.serializeMsgs(s.newMsgSend(gmpAddr, recipient))

	_, err := s.chainA.GetSimApp().GMPKeeper.OnRecvPacket(
		s.chainA.GetContext(),
		&data,
		ibctesting.FirstClientID,
	)
	s.Require().NoError(err)
}
