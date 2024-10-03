package api_test

import (
	testifysuite "github.com/stretchr/testify/suite"
	"testing"
)

type ApiTestSuite struct {
	testifysuite.Suite
}

func TestApiTestSuite(t *testing.T) {
	testifysuite.Run(t, new(ApiTestSuite))
}
