package keeper_test

import (
	sdkmath "cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))
}

type amountType int

const (
	escrow amountType = iota
	balance
)

func (suite *KeeperTestSuite) assertAmountOnChain(chain *ibctesting.TestChain, balanceType amountType, amount sdkmath.Int, denom string) {
	var total sdk.Coin
	switch balanceType {
	case escrow:
		total = chain.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(chain.GetContext(), denom)
		totalV2 := chain.GetSimApp().TransferKeeperV2.GetTotalEscrowForDenom(chain.GetContext(), denom)
		suite.Require().Equal(total, totalV2, "escrow balance mismatch")
	case balance:
		total = chain.GetSimApp().BankKeeper.GetBalance(chain.GetContext(), chain.SenderAccounts[0].SenderAccount.GetAddress(), denom)
	default:
		suite.Fail("invalid amountType %s", balanceType)
	}
	suite.Require().Equal(amount, total.Amount, fmt.Sprintf("Chain %s: got balance of %s, wanted %s", chain.Name(), total.Amount.String(), amount.String()))
}

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}
