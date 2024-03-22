package celestia_test

import (
	fmt "fmt"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibccelestia "github.com/cosmos/ibc-go/v8/modules/light-clients/07-celestia"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (suite *CelestiaTestSuite) TestStatus() {
	var clientID string

	testCases := []struct {
		name      string
		malleate  func()
		expStatus exported.Status
	}{
		{
			"success",
			func() {},
			exported.Active,
		},
		{
			"client state not found",
			func() {
				clientID = fmt.Sprintf("%s-%d", ibccelestia.ModuleName, 100)
			},
			exported.Unknown,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			path := ibctesting.NewPath(suite.chainA, suite.chainB)

			clientID = suite.CreateClient(path.EndpointA)
			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			tc.malleate()

			status := lightClientModule.Status(suite.chainA.GetContext(), clientID)
			suite.Require().Equal(tc.expStatus, status)
		})
	}
}
