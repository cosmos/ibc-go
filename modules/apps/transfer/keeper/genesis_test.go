package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

func (s *KeeperTestSuite) TestGenesis() {
	getTrace := func(index uint) string {
		return fmt.Sprintf("transfer/channelToChain%d", index)
	}

	var (
		traces                types.Traces
		escrows               sdk.Coins
		pathsAndEscrowAmounts = []struct {
			path   string
			escrow string
		}{
			{getTrace(0), "10"},
			{fmt.Sprintf("%s/%s", getTrace(1), getTrace(0)), "100000"},
			{fmt.Sprintf("%s/%s/%s", getTrace(2), getTrace(1), getTrace(0)), "10000000000"},
			{fmt.Sprintf("%s/%s/%s/%s", getTrace(3), getTrace(2), getTrace(1), getTrace(0)), "1000000000000000"},
			{fmt.Sprintf("%s/%s/%s/%s/%s", getTrace(4), getTrace(3), getTrace(2), getTrace(1), getTrace(0)), "100000000000000000000"},
		}
	)

	for _, pathAndEscrowAmount := range pathsAndEscrowAmounts {
		denomTrace := types.DenomTrace{
			BaseDenom: "uatom",
			Path:      pathAndEscrowAmount.path,
		}
		traces = append(types.Traces{denomTrace}, traces...)
		s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(s.chainA.GetContext(), denomTrace)

		denom := denomTrace.IBCDenom()
		amount, ok := sdkmath.NewIntFromString(pathAndEscrowAmount.escrow)
		s.Require().True(ok)
		escrows = append(sdk.NewCoins(sdk.NewCoin(denom, amount)), escrows...)
		s.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(s.chainA.GetContext(), sdk.NewCoin(denom, amount))
	}

	genesis := s.chainA.GetSimApp().TransferKeeper.ExportGenesis(s.chainA.GetContext())

	s.Require().Equal(types.PortID, genesis.PortId)
	s.Require().Equal(traces.Sort(), genesis.DenomTraces)
	s.Require().Equal(escrows.Sort(), genesis.TotalEscrowed)

	s.Require().NotPanics(func() {
		s.chainA.GetSimApp().TransferKeeper.InitGenesis(s.chainA.GetContext(), *genesis)
	})
}
