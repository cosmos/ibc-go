package celestia_test

import (
	fmt "fmt"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	celestia "github.com/cosmos/ibc-go/v8/modules/light-clients/07-celestia"
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
				clientID = fmt.Sprintf("%s-%d", celestia.ModuleName, 100)
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

// func (suite *SoloMachineTestSuite) TestLcmLatestHeight() {
// 	var clientID string

// 	// test singlesig and multisig public keys
// 	for _, sm := range []*ibctesting.Solomachine{suite.solomachine, suite.solomachineMulti} {
// 		testCases := []struct {
// 			name      string
// 			malleate  func()
// 			expHeight exported.Height
// 		}{
// 			{
// 				"success",
// 				func() {},
// 				clienttypes.NewHeight(0, 1),
// 			},
// 			{
// 				"client state not found",
// 				func() {
// 					clientID = fmt.Sprintf("%s-%d", exported.Solomachine, 100)
// 				},
// 				clienttypes.ZeroHeight(),
// 			},
// 		}

// 		for _, tc := range testCases {
// 			tc := tc

// 			suite.Run(tc.name, func() {
// 				clientID = sm.ClientID
// 				clientState := sm.ClientState()

// 				suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(suite.chainA.GetContext(), clientID, clientState)

// 				lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
// 				suite.Require().True(found)

// 				tc.malleate()

// 				height := lightClientModule.LatestHeight(suite.chainA.GetContext(), clientID)
// 				suite.Require().Equal(tc.expHeight, height)
// 			})
// 		}
// 	}
// }
