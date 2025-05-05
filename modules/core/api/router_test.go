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
			name: "success: prefix based routing works",
			malleate: func() {
				router.AddRoute("somemodule", &mockv2.IBCModule{})
				router.AddRoute("port01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				suite.Require().True(router.HasRoute("somemodule"))
				suite.Require().True(router.HasRoute("somemoduleport01"))
				ok, prefix := router.HasRoute("somemoduleport01")
				suite.Require().Equal(true, ok)
				suite.Require().Equal("somemodule", prefix)
				suite.Require().NotNil(router.Route("somemoduleport01"))
				suite.Require().True(router.HasRoute("port01"))
			},
		},
		{
			name: "failure: panics on duplicate module",
			malleate: func() {
				router.AddRoute("port01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				suite.Require().PanicsWithError("route port01 has already been covered by registered prefix: port01", func() {
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
		{
			name:     "failure: panics conflicting routes registered",
			malleate: func() {},
			assertionFn: func() {
				suite.Require().PanicsWithError("route someModuleWithSpecificPath has already been covered by registered prefix: someModule", func() {
					router.AddRoute("someModule", &mockv2.IBCModule{})
					router.AddRoute("someModuleWithSpecificPath", &mockv2.IBCModule{})
				})
			},
		},
		{
			name:     "failure: panics conflicting routes registered, when shorter prefix is added",
			malleate: func() {},
			assertionFn: func() {
				suite.Require().PanicsWithError("route someLonger is a prefix for already registered route: someLongerPrefixModule", func() {
					router.AddRoute("someLongerPrefixModule", &mockv2.IBCModule{})
					router.AddRoute("someLonger", &mockv2.IBCModule{})
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
