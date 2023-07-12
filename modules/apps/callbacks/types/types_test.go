package types_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

// CallbacksTestSuite defines the needed instances and methods to test callbacks
type CallbacksTypesTestSuite struct {
	suite.Suite

	coord *ibctesting.Coordinator

	chain *ibctesting.TestChain
}

// SetupTest creates a coordinator with 1 test chain.
func (suite *CallbacksTypesTestSuite) SetupSuite() {
	suite.coord = ibctesting.NewCoordinator(suite.T(), 1)
	suite.chain = suite.coord.GetChain(ibctesting.GetChainID(1))
}

func TestCallbacksTypesTestSuite(t *testing.T) {
	suite.Run(t, new(CallbacksTypesTestSuite))
}

type MockPacketDataUnmarshaler struct{}

func (m MockPacketDataUnmarshaler) UnmarshalPacketData(data []byte) (interface{}, error) {
	if reflect.DeepEqual(data, []byte("no unmarshaler error")) {
		return nil, nil
	}
	return nil, fmt.Errorf("mock error")
}
