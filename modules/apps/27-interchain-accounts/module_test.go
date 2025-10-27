package ica_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type InterchainAccountsTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator
}

func TestICATestSuite(t *testing.T) {
	testifysuite.Run(t, new(InterchainAccountsTestSuite))
}

func (s *InterchainAccountsTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
}
