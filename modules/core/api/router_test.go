package api_test

import (
	"github.com/cosmos/ibc-go/v10/modules/core/api"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

func (suite *APITestSuite) TestRouter() {
	var router *api.Router

	testCases := []struct {
		name        string
		malleate    func()
		assertionFn func()
	}{
		{
			name: "success",
			malleate: func() {
				router.AddRoute("port01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				suite.Require().True(router.HasRoute("port01"))
			},
		},
		{
			name: "success: multiple modules",
			malleate: func() {
				router.AddRoute("port01", &mockv2.IBCModule{})
				router.AddRoute("port02", &mockv2.IBCModule{})
				router.AddRoute("port03", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				suite.Require().True(router.HasRoute("port01"))
				suite.Require().True(router.HasRoute("port02"))
				suite.Require().True(router.HasRoute("port03"))
			},
		},
		{
			name: "failure: panics on duplicate module",
			malleate: func() {
				router.AddRoute("port01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				suite.Require().PanicsWithError("route port01 has already been registered", func() {
					router.AddRoute("port01", &mockv2.IBCModule{})
				})
			},
		},
		{
			name:     "failure: panics invalid-name",
			malleate: func() {},
			assertionFn: func() {
				suite.Require().PanicsWithError("route expressions can only contain alphanumeric characters", func() {
					router.AddRoute("port-02", &mockv2.IBCModule{})
				})
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			router = api.NewRouter()

			tc.malleate()

			tc.assertionFn()
		})
	}
}
