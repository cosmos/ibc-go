package types_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type BeefyTestSuite struct {
	suite.Suite
}

func (suite *BeefyTestSuite) SetupTest() {

}

func TestSoloMachineTestSuite(t *testing.T) {
	suite.Run(t, new(BeefyTestSuite))
}
