//go:build !test_e2e

package transfer

import (
	"context"
	"testing"

	"github.com/cosmos/interchaintest/v10/ibc"
	test "github.com/cosmos/interchaintest/v10/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

// compatibility:from_version: v7.10.0
func TestAuthzTransferTestSuite(t *testing.T) {
	testifysuite.Run(t, new(AuthzTransferTestSuite))
}

type AuthzTransferTestSuite struct {
	testsuite.E2ETestSuite
}

// SetupSuite sets up chains for the current test suite
func (s *AuthzTransferTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil)
}

// QueryGranterGrants returns all GrantAuthorizations for the given granterAddress.
func (*AuthzTransferTestSuite) QueryGranterGrants(ctx context.Context, chain ibc.Chain, granterAddress string) ([]*authz.GrantAuthorization, error) {
	res, err := query.GRPCQuery[authz.QueryGranterGrantsResponse](ctx, chain, &authz.QueryGranterGrantsRequest{
		Granter: granterAddress,
	})
	if err != nil {
		return nil, err
	}

	return res.Grants, nil
}

func (s *AuthzTransferTestSuite) TestAuthz_MsgTransfer_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	t.Parallel()

	chainA, chainB := s.GetChains()
	chainADenom := chainA.Config().Denom

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	granterWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	granterAddress := granterWallet.FormattedAddress()

	granteeWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	granteeAddress := granteeWallet.FormattedAddress()

	receiverWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	receiverWalletAddress := receiverWallet.FormattedAddress()

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	// createMsgGrantFn initializes a TransferAuthorization and broadcasts a MsgGrant message.
	createMsgGrantFn := func(t *testing.T) {
		t.Helper()
		transferAuth := transfertypes.TransferAuthorization{
			Allocations: []transfertypes.Allocation{
				{
					SourcePort:    channelA.PortID,
					SourceChannel: channelA.ChannelID,
					SpendLimit:    sdk.NewCoins(sdk.NewCoin(chainADenom, sdkmath.NewInt(testvalues.StartingTokenAmount))),
					AllowList:     []string{receiverWalletAddress},
				},
			},
		}

		protoAny, err := codectypes.NewAnyWithValue(&transferAuth)
		s.Require().NoError(err)

		msgGrant := &authz.MsgGrant{
			Granter: granterAddress,
			Grantee: granteeAddress,
			Grant: authz.Grant{
				Authorization: protoAny,
				// no expiration
				Expiration: nil,
			},
		}

		resp := s.BroadcastMessages(t.Context(), chainA, granterWallet, msgGrant)
		s.AssertTxSuccess(resp)
	}

	// verifyGrantFn returns a test function which asserts chainA has a grant authorization
	// with the given spend limit.
	verifyGrantFn := func(expectedLimit int64) func(t *testing.T) {
		t.Helper()
		return func(t *testing.T) {
			t.Helper()
			grantAuths, err := s.QueryGranterGrants(ctx, chainA, granterAddress)

			s.Require().NoError(err)
			s.Require().Len(grantAuths, 1)
			grantAuthorization := grantAuths[0]

			transferAuth := s.extractTransferAuthorizationFromGrantAuthorization(grantAuthorization)
			expectedSpendLimit := sdk.NewCoins(sdk.NewCoin(chainADenom, sdkmath.NewInt(expectedLimit)))
			s.Require().Equal(expectedSpendLimit, transferAuth.Allocations[0].SpendLimit)
		}
	}

	t.Run("broadcast MsgGrant", createMsgGrantFn)

	t.Run("broadcast MsgExec for ibc MsgTransfer", func(t *testing.T) {
		transferMsg := testsuite.GetMsgTransfer(
			channelA.PortID,
			channelA.ChannelID,
			channelA.Version,
			testvalues.DefaultTransferAmount(chainADenom),
			granterAddress,
			receiverWalletAddress,
			s.GetTimeoutHeight(ctx, chainB),
			0,
			"",
		)

		protoAny, err := codectypes.NewAnyWithValue(transferMsg)
		s.Require().NoError(err)

		msgExec := &authz.MsgExec{
			Grantee: granteeAddress,
			Msgs:    []*codectypes.Any{protoAny},
		}

		resp := s.BroadcastMessages(t.Context(), chainA, granteeWallet, msgExec)
		s.AssertTxSuccess(resp)
	})

	t.Run("verify granter wallet amount", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, granterWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	s.Require().NoError(test.WaitForBlocks(context.TODO(), 10, chainB))

	t.Run("verify receiver wallet amount", func(t *testing.T) {
		chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)
		actualBalance, err := query.Balance(ctx, chainB, receiverWalletAddress, chainBIBCToken.IBCDenom())

		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, actualBalance.Int64())
	})

	t.Run("granter grant spend limit reduced", verifyGrantFn(testvalues.StartingTokenAmount-testvalues.IBCTransferAmount))

	t.Run("re-initialize MsgGrant", createMsgGrantFn)

	t.Run("granter grant was reinitialized", verifyGrantFn(testvalues.StartingTokenAmount))

	t.Run("revoke access", func(t *testing.T) {
		msgRevoke := authz.MsgRevoke{
			Granter:    granterAddress,
			Grantee:    granteeAddress,
			MsgTypeUrl: (*transfertypes.TransferAuthorization)(nil).MsgTypeURL(),
		}

		resp := s.BroadcastMessages(t.Context(), chainA, granterWallet, &msgRevoke)
		s.AssertTxSuccess(resp)
	})

	t.Run("exec unauthorized MsgTransfer", func(t *testing.T) {
		transferMsg := testsuite.GetMsgTransfer(
			channelA.PortID,
			channelA.ChannelID,
			channelA.Version,
			testvalues.DefaultTransferAmount(chainADenom),
			granterAddress,
			receiverWalletAddress,
			s.GetTimeoutHeight(ctx, chainB),
			0,
			"",
		)

		protoAny, err := codectypes.NewAnyWithValue(transferMsg)
		s.Require().NoError(err)

		msgExec := &authz.MsgExec{
			Grantee: granteeAddress,
			Msgs:    []*codectypes.Any{protoAny},
		}

		resp := s.BroadcastMessages(t.Context(), chainA, granteeWallet, msgExec)
		s.AssertTxFailure(resp, authz.ErrNoAuthorizationFound)
	})
}

func (s *AuthzTransferTestSuite) TestAuthz_InvalidTransferAuthorizations() {
	t := s.T()
	ctx := context.TODO()

	testName := t.Name()
	t.Parallel()

	chainA, chainB := s.GetChains()

	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

	chainADenom := chainA.Config().Denom
	chainAVersion := chainA.Config().Images[0].Version

	granterWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	granterAddress := granterWallet.FormattedAddress()

	granteeWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	granteeAddress := granteeWallet.FormattedAddress()

	receiverWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	receiverWalletAddress := receiverWallet.FormattedAddress()

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(relayer, testName)
	})

	const spendLimit = 1000

	t.Run("broadcast MsgGrant", func(t *testing.T) {
		transferAuth := transfertypes.TransferAuthorization{
			Allocations: []transfertypes.Allocation{
				{
					SourcePort:    channelA.PortID,
					SourceChannel: channelA.ChannelID,
					SpendLimit:    sdk.NewCoins(sdk.NewCoin(chainADenom, sdkmath.NewInt(spendLimit))),
					AllowList:     []string{receiverWalletAddress},
				},
			},
		}

		protoAny, err := codectypes.NewAnyWithValue(&transferAuth)
		s.Require().NoError(err)

		msgGrant := &authz.MsgGrant{
			Granter: granterAddress,
			Grantee: granteeAddress,
			Grant: authz.Grant{
				Authorization: protoAny,
				// no expiration
				Expiration: nil,
			},
		}

		resp := s.BroadcastMessages(t.Context(), chainA, granterWallet, msgGrant)
		s.AssertTxSuccess(resp)
	})

	t.Run("exceed spend limit", func(t *testing.T) {
		const invalidSpendAmount = spendLimit + 1

		t.Run("broadcast MsgExec for ibc MsgTransfer", func(t *testing.T) {
			transferMsg := testsuite.GetMsgTransfer(
				channelA.PortID,
				channelA.ChannelID,
				channelA.Version,
				sdk.Coin{Denom: chainADenom, Amount: sdkmath.NewInt(invalidSpendAmount)},
				granterAddress,
				receiverWalletAddress,
				s.GetTimeoutHeight(ctx, chainB),
				0,
				"",
			)

			protoAny, err := codectypes.NewAnyWithValue(transferMsg)
			s.Require().NoError(err)

			msgExec := &authz.MsgExec{
				Grantee: granteeAddress,
				Msgs:    []*codectypes.Any{protoAny},
			}

			resp := s.BroadcastMessages(t.Context(), chainA, granteeWallet, msgExec)
			if testvalues.IbcErrorsFeatureReleases.IsSupported(chainAVersion) {
				s.AssertTxFailure(resp, ibcerrors.ErrInsufficientFunds)
			} else {
				s.AssertTxFailure(resp, sdkerrors.ErrInsufficientFunds)
			}
		})

		t.Run("verify granter wallet amount", func(t *testing.T) {
			actualBalance, err := s.GetChainANativeBalance(ctx, granterWallet)
			s.Require().NoError(err)
			s.Require().Equal(testvalues.StartingTokenAmount, actualBalance)
		})

		t.Run("verify receiver wallet amount", func(t *testing.T) {
			chainBIBCToken := testsuite.GetIBCToken(chainADenom, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID)
			actualBalance, err := query.Balance(ctx, chainB, receiverWalletAddress, chainBIBCToken.IBCDenom())

			s.Require().NoError(err)
			s.Require().Equal(int64(0), actualBalance.Int64())
		})

		t.Run("granter grant spend limit unchanged", func(t *testing.T) {
			grantAuths, err := s.QueryGranterGrants(ctx, chainA, granterAddress)

			s.Require().NoError(err)
			s.Require().Len(grantAuths, 1)
			grantAuthorization := grantAuths[0]

			transferAuth := s.extractTransferAuthorizationFromGrantAuthorization(grantAuthorization)
			expectedSpendLimit := sdk.NewCoins(sdk.NewCoin(chainADenom, sdkmath.NewInt(spendLimit)))
			s.Require().Equal(expectedSpendLimit, transferAuth.Allocations[0].SpendLimit)
		})
	})

	t.Run("send funds to invalid address", func(t *testing.T) {
		invalidWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
		invalidWalletAddress := invalidWallet.FormattedAddress()

		t.Run("broadcast MsgExec for ibc MsgTransfer", func(t *testing.T) {
			transferMsg := testsuite.GetMsgTransfer(
				channelA.PortID,
				channelA.ChannelID,
				channelA.Version,
				sdk.Coin{Denom: chainADenom, Amount: sdkmath.NewInt(spendLimit)},
				granterAddress,
				invalidWalletAddress,
				s.GetTimeoutHeight(ctx, chainB),
				0,
				"",
			)

			protoAny, err := codectypes.NewAnyWithValue(transferMsg)
			s.Require().NoError(err)

			msgExec := &authz.MsgExec{
				Grantee: granteeAddress,
				Msgs:    []*codectypes.Any{protoAny},
			}

			resp := s.BroadcastMessages(t.Context(), chainA, granteeWallet, msgExec)
			if testvalues.IbcErrorsFeatureReleases.IsSupported(chainAVersion) {
				s.AssertTxFailure(resp, ibcerrors.ErrInvalidAddress)
			} else {
				s.AssertTxFailure(resp, sdkerrors.ErrInvalidAddress)
			}
		})
	})
}

// extractTransferAuthorizationFromGrantAuthorization extracts a TransferAuthorization from the given
// GrantAuthorization.
func (s *AuthzTransferTestSuite) extractTransferAuthorizationFromGrantAuthorization(grantAuth *authz.GrantAuthorization) *transfertypes.TransferAuthorization {
	cfg := testsuite.SDKEncodingConfig()
	var authorization authz.Authorization
	err := cfg.InterfaceRegistry.UnpackAny(grantAuth.Authorization, &authorization)
	s.Require().NoError(err)

	transferAuth, ok := authorization.(*transfertypes.TransferAuthorization)
	s.Require().True(ok)
	return transferAuth
}
