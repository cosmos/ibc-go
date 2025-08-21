package api_test

import (
	"github.com/cosmos/ibc-go/v10/modules/core/api"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

func (s *APITestSuite) TestRouter() {
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
				s.Require().True(router.HasRoute("port01"))
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
				s.Require().True(router.HasRoute("port01"))
				s.Require().True(router.HasRoute("port02"))
				s.Require().True(router.HasRoute("port03"))
			},
		},
		{
			name: "success: prefix based routing works",
			malleate: func() {
				router.AddPrefixRoute("somemodule", &mockv2.IBCModule{})
				router.AddRoute("port01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				s.Require().True(router.HasRoute("somemodule"))
				s.Require().True(router.HasRoute("somemoduleport01"))
				s.Require().NotNil(router.Route("somemoduleport01"))
				s.Require().True(router.HasRoute("port01"))
			},
		},
		{
			name: "failure: panics on adding direct route after overlapping prefix route",
			malleate: func() {
				router.AddPrefixRoute("someModule", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				s.Require().PanicsWithError("route someModuleWithSpecificPath is already matched by registered prefix route: someModule", func() {
					router.AddRoute("someModuleWithSpecificPath", &mockv2.IBCModule{})
				})
			},
		},
		{
			name: "failure: panics on adding prefix route after overlapping direct route",
			malleate: func() {
				router.AddRoute("someModuleWithSpecificPath", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				s.Require().PanicsWithError("route prefix someModule is a prefix for already registered route: someModuleWithSpecificPath", func() {
					router.AddPrefixRoute("someModule", &mockv2.IBCModule{})
				})
			},
		},
		{
			name: "failure: panics on duplicate route",
			malleate: func() {
				router.AddRoute("port01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				s.Require().PanicsWithError("route port01 has already been registered", func() {
					router.AddRoute("port01", &mockv2.IBCModule{})
				})
			},
		},
		{
			name: "failure: panics on duplicate route / prefix route",
			malleate: func() {
				router.AddRoute("port01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				s.Require().PanicsWithError("route prefix port01 is a prefix for already registered route: port01", func() {
					router.AddPrefixRoute("port01", &mockv2.IBCModule{})
				})
			},
		},
		{
			name: "failure: panics on duplicate prefix route",
			malleate: func() {
				router.AddPrefixRoute("port01", &mockv2.IBCModule{})
			},
			assertionFn: func() {
				s.Require().PanicsWithError("route prefix port01 has already been covered by registered prefix: port01", func() {
					router.AddPrefixRoute("port01", &mockv2.IBCModule{})
				})
			},
		},
		{
			name:     "failure: panics invalid-name",
			malleate: func() {},
			assertionFn: func() {
				s.Require().PanicsWithError("route expressions can only contain alphanumeric characters", func() {
					router.AddRoute("port-02", &mockv2.IBCModule{})
				})
			},
		},
		{
			name:     "failure: panics conflicting prefix routes registered, when shorter prefix is added",
			malleate: func() {},
			assertionFn: func() {
				s.Require().PanicsWithError("route prefix someLonger is a prefix for already registered prefix: someLongerPrefixModule", func() {
					router.AddPrefixRoute("someLongerPrefixModule", &mockv2.IBCModule{})
					router.AddPrefixRoute("someLonger", &mockv2.IBCModule{})
				})
			},
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			router = api.NewRouter()

			tc.malleate()

			tc.assertionFn()
		})
	}
}
