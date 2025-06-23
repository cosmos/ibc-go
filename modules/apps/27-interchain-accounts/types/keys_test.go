package types_test

import (
	"fmt"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *TypesTestSuite) TestKeyActiveChannel() {
	key := types.KeyActiveChannel("port-id", "connection-id")
	s.Require().Equal("activeChannel/port-id/connection-id", string(key))
}

func (s *TypesTestSuite) TestKeyOwnerAccount() {
	key := types.KeyOwnerAccount("port-id", "connection-id")
	s.Require().Equal("owner/port-id/connection-id", string(key))
}

func (s *TypesTestSuite) TestKeyIsMiddlewareEnabled() {
	key := types.KeyIsMiddlewareEnabled(ibctesting.MockPort, ibctesting.FirstChannelID)
	s.Require().Equal(fmt.Sprintf("%s/%s/%s", types.IsMiddlewareEnabledPrefix, ibctesting.MockPort, ibctesting.FirstChannelID), string(key))
}
