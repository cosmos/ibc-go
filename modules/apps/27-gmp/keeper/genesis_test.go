// SPDX-License-Identifier: Apache-2.0

package keeper_test

import (
	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/27-gmp/types"
	ibcerrors "github.com/cosmos/ibc-go/v11/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
)

const invalid = "invalid"

func (s *KeeperTestSuite) TestInitGenesis() {
	var genesisState *types.GenesisState

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
			"success: empty genesis",
			func() {
				genesisState = types.DefaultGenesisState()
			},
			nil,
		},
		{
			"failure: invalid account address",
			func() {
				genesisState.Ics27Accounts[0].AccountAddress = invalid
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"success: sender is a counterparty (non-bech32) identifier",
			func() {
				genesisState.Ics27Accounts[0].AccountId.Sender = "0x1234567890abcdef1234567890abcdef12345678"
			},
			nil,
		},
		{
			"failure: empty sender address",
			func() {
				genesisState.Ics27Accounts[0].AccountId.Sender = " "
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: sender address exceeds max length",
			func() {
				genesisState.Ics27Accounts[0].AccountId.Sender = ibctesting.GenerateString(types.MaximumSenderLength + 1)
			},
			ibcerrors.ErrInvalidAddress,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			sender := s.chainB.SenderAccount.GetAddress().String()
			accountID := types.NewAccountIdentifier(ibctesting.FirstClientID, sender, []byte(testSalt))
			addr, err := types.BuildAddressPredictable(&accountID)
			s.Require().NoError(err)

			genesisState = &types.GenesisState{
				Ics27Accounts: []types.RegisteredICS27Account{
					{
						AccountAddress: addr.String(),
						AccountId:      accountID,
					},
				},
			}

			tc.malleate()

			err = s.chainA.GetSimApp().GMPKeeper.InitGenesis(s.chainA.GetContext(), genesisState)

			if tc.expErr == nil {
				s.Require().NoError(err)

				if len(genesisState.Ics27Accounts) > 0 {
					account := genesisState.Ics27Accounts[0]
					storedAddr, err := s.chainA.GetSimApp().GMPKeeper.GetOrComputeICS27Address(
						s.chainA.GetContext(),
						&account.AccountId,
					)
					s.Require().NoError(err)
					s.Require().Equal(account.AccountAddress, storedAddr)
					storedAccount, err := s.chainA.GetSimApp().GMPKeeper.Accounts.Get(
						s.chainA.GetContext(),
						collections.Join3(account.AccountId.ClientId, account.AccountId.Sender, account.AccountId.Salt),
					)
					s.Require().NoError(err)
					s.Require().Equal(account.AccountAddress, storedAccount.Address)
					addressFromMap, err := s.chainA.GetSimApp().GMPKeeper.AccountsByAddress.Get(
						s.chainA.GetContext(),
						sdk.MustAccAddressFromBech32(account.AccountAddress),
					)
					s.Require().NoError(err)
					s.Require().Equal(account.AccountAddress, addressFromMap.Address)
					s.Require().Equal(account.AccountId.ClientId, storedAccount.AccountId.ClientId)
					s.Require().Equal(account.AccountId.Sender, storedAccount.AccountId.Sender)
					s.Require().Equal(account.AccountId.Salt, storedAccount.AccountId.Salt)
				}
			} else {
				s.Require().ErrorIs(err, tc.expErr)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetAuthority() {
	s.SetupTest()

	authority := s.chainA.GetSimApp().GMPKeeper.GetAuthority()
	s.Require().NotEmpty(authority)
}

func (s *KeeperTestSuite) TestExportGenesis() {
	s.SetupTest()

	sender := s.chainB.SenderAccount.GetAddress().String()
	accountID := types.NewAccountIdentifier(ibctesting.FirstClientID, sender, []byte(testSalt))
	addr, err := types.BuildAddressPredictable(&accountID)
	s.Require().NoError(err)
	gmpAccountAddr := addr.String()

	s.createGMPAccount(gmpAccountAddr)

	genesisState, err := s.chainA.GetSimApp().GMPKeeper.ExportGenesis(s.chainA.GetContext())
	s.Require().NoError(err)
	s.Require().Len(genesisState.Ics27Accounts, 1)
	s.Require().Equal(gmpAccountAddr, genesisState.Ics27Accounts[0].AccountAddress)
	s.Require().Equal(ibctesting.FirstClientID, genesisState.Ics27Accounts[0].AccountId.ClientId)
	s.Require().Equal(sender, genesisState.Ics27Accounts[0].AccountId.Sender)
	s.Require().Equal([]byte(testSalt), genesisState.Ics27Accounts[0].AccountId.Salt)
}

// TestGenesisRoundTripRemoteSender ensures that an ICS27 account created for a
// sender that is not a local bech32 address (e.g. an EVM hex address) survives
// an export -> validate -> import genesis round trip.
func (s *KeeperTestSuite) TestGenesisRoundTripRemoteSender() {
	s.SetupTest()
	ctx := s.chainA.GetContext()
	gmpKeeper := s.chainA.GetSimApp().GMPKeeper

	evmSender := "0x1234567890abcdef1234567890abcdef12345678"
	packetData := types.NewGMPPacketData(evmSender, "", []byte(testSalt), []byte{}, "")

	// account creation happens before the (expected) payload execution failure
	_, err := gmpKeeper.OnRecvPacket(ctx, &packetData, ibctesting.FirstClientID)
	s.Require().ErrorIs(err, types.ErrInvalidPayload)

	exported, err := gmpKeeper.ExportGenesis(ctx)
	s.Require().NoError(err)
	s.Require().Len(exported.Ics27Accounts, 1)
	s.Require().Equal(evmSender, exported.Ics27Accounts[0].AccountId.Sender)

	s.Require().NoError(exported.Validate())
	s.Require().NoError(gmpKeeper.InitGenesis(ctx, exported))
}
