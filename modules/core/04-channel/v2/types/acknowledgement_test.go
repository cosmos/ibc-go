package types_test

import (
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

// Test_ValidateAcknowledgement tests the acknowledgements Validate method
func (s *TypesTestSuite) Test_ValidateAcknowledgement() {
	testCases := []struct {
		name     string
		ack      types.Acknowledgement
		expError error
	}{
		{
			"success: valid successful ack",
			types.NewAcknowledgement([]byte("appAck1")),
			nil,
		},
		{
			"success: valid failed ack",
			types.NewAcknowledgement(types.ErrorAcknowledgement[:]),
			nil,
		},
		{
			"success: more than one app acknowledgements",
			types.NewAcknowledgement([]byte("appAck1"), []byte("appAck2")),
			nil,
		},
		{
			"failure: empty acknowledgement",
			types.NewAcknowledgement(),
			types.ErrInvalidAcknowledgement,
		},
		{
			"failure: app acknowledgement is empty",
			types.NewAcknowledgement([]byte("")),
			types.ErrInvalidAcknowledgement,
		},
		{
			"failure: error acknowledgment in multiple payload list",
			types.NewAcknowledgement(types.ErrorAcknowledgement[:], []byte("appAck2")),
			types.ErrInvalidAcknowledgement,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			err := tc.ack.Validate()

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err)
			} else {
				s.Require().ErrorIs(err, tc.expError)
			}
		})
	}
}
