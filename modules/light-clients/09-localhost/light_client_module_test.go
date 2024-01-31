package localhost_test

import (
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

func (s *LocalhostTestSuite) TestStatus() {
	lightClientModule, found := s.chain.GetSimApp().IBCKeeper.ClientKeeper.GetRouter().GetRoute(exported.Localhost)
	s.Require().True(found)
	s.Require().Equal(exported.Active, lightClientModule.Status(s.chain.GetContext(), exported.LocalhostClientID))
}
