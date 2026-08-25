// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"
)

type APITestSuite struct {
	testifysuite.Suite
}

func TestApiTestSuite(t *testing.T) {
	testifysuite.Run(t, new(APITestSuite))
}
