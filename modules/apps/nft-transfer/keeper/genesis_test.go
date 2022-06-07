package keeper_test

import (
	"fmt"

	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
)

func (suite *KeeperTestSuite) TestGenesis() {
	var (
		path   string
		traces types.Traces
	)

	for i := 0; i < 5; i++ {
		prefix := fmt.Sprintf("nft-transfer/channelToChain%d", i)
		if i == 0 {
			path = prefix
		} else {
			path = prefix + "/" + path
		}

		classTrace := types.ClassTrace{
			BaseClassId: "kitty",
			Path:        path,
		}
		traces = append(types.Traces{classTrace}, traces...)
		suite.chainA.GetSimApp().NFTTransferKeeper.SetClassTrace(suite.chainA.GetContext(), classTrace)
	}

	genesis := suite.chainA.GetSimApp().NFTTransferKeeper.ExportGenesis(suite.chainA.GetContext())

	suite.Require().Equal(types.PortID, genesis.PortId)
	suite.Require().Equal(traces.Sort(), genesis.Traces)

	suite.Require().NotPanics(func() {
		suite.chainA.GetSimApp().NFTTransferKeeper.InitGenesis(suite.chainA.GetContext(), *genesis)
	})
}
