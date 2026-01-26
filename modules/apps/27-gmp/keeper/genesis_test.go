package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

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
				genesisState.Ics27Accounts[0].AccountAddress = "invalid"
			},
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: invalid sender address",
			func() {
				genesisState.Ics27Accounts[0].AccountId.Sender = "invalid"
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
						AccountAddress: sdk.AccAddress(addr).String(),
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
	gmpAccountAddr := sdk.AccAddress(addr).String()

	s.createGMPAccount(gmpAccountAddr)

	genesisState, err := s.chainA.GetSimApp().GMPKeeper.ExportGenesis(s.chainA.GetContext())
	s.Require().NoError(err)
	s.Require().Len(genesisState.Ics27Accounts, 1)
	s.Require().Equal(gmpAccountAddr, genesisState.Ics27Accounts[0].AccountAddress)
	s.Require().Equal(ibctesting.FirstClientID, genesisState.Ics27Accounts[0].AccountId.ClientId)
	s.Require().Equal(sender, genesisState.Ics27Accounts[0].AccountId.Sender)
	s.Require().Equal([]byte(testSalt), genesisState.Ics27Accounts[0].AccountId.Salt)
}
