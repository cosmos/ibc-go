package api_test

import (
	"github.com/cosmos/ibc-go/v9/modules/core/api"
	mockv2 "github.com/cosmos/ibc-go/v9/testing/mock/v2"
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
				router.AddRoute("module01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				suite.Require().True(router.HasRoute("module01"))
			},
		},
		{
			name: "success: multiple modules",
			malleate: func() {
				router.AddRoute("module01", &mockv2.IBCModule{})
				router.AddRoute("module02", &mockv2.IBCModule{})
				router.AddRoute("module03", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				suite.Require().True(router.HasRoute("module01"))
				suite.Require().True(router.HasRoute("module02"))
				suite.Require().True(router.HasRoute("module03"))
			},
		},
		{
			name: "success: find by prefix",
			malleate: func() {
				router.AddRoute("module01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				suite.Require().True(router.HasRoute("module01-foo"))
			},
		},
		{
			name: "failure: panics on duplicate module",
			malleate: func() {
				router.AddRoute("module01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				suite.Require().PanicsWithError("route module01 has already been registered", func() {
					router.AddRoute("module01", &mockv2.IBCModule{})
				})
			},
		},
		{
			name: "failure: panics invalid-name",
			malleate: func() {
				router.AddRoute("module01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				suite.Require().PanicsWithError("route expressions can only contain alphanumeric characters", func() {
					router.AddRoute("module-02", &mockv2.IBCModule{})
				})
			},
		},
	}
	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			router = api.NewRouter()

			tc.malleate()

			tc.assertionFn()
		})
	}
}
