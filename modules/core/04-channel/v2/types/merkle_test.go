package types_test

import (
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (s *TypesTestSuite) TestBuildMerklePath() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupV2()

	prefixPath := [][]byte{[]byte("ibc"), []byte("")}
	packetCommitmentKey := host.PacketCommitmentKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1)
	emptyPrefixPanicMsg := "cannot build merkle path with empty prefix"

	testCases := []struct {
		name      string
		prefix    [][]byte
		path      []byte
		expPath   commitmenttypesv2.MerklePath
		expPanics *string
	}{
		{
			name:    "standard ibc path",
			prefix:  prefixPath,
			path:    packetCommitmentKey,
			expPath: commitmenttypesv2.NewMerklePath([]byte("ibc"), packetCommitmentKey),
		},
		{
			name:    "non-empty last element prefix path",
			prefix:  [][]byte{[]byte("ibc"), []byte("abc")},
			path:    packetCommitmentKey,
			expPath: commitmenttypesv2.NewMerklePath([]byte("ibc"), append([]byte("abc"), packetCommitmentKey...)),
		},
		{
			name:    "many elements in prefix path",
			prefix:  [][]byte{[]byte("ibc"), []byte("a"), []byte("b"), []byte("c"), []byte("d")},
			path:    packetCommitmentKey,
			expPath: commitmenttypesv2.NewMerklePath([]byte("ibc"), []byte("a"), []byte("b"), []byte("c"), append([]byte("d"), packetCommitmentKey...)),
		},
		{
			name:      "empty prefix",
			prefix:    [][]byte{},
			path:      packetCommitmentKey,
			expPanics: &emptyPrefixPanicMsg,
		},
		{
			name:    "empty path",
			prefix:  prefixPath,
			path:    []byte{},
			expPath: commitmenttypesv2.NewMerklePath([]byte("ibc"), []byte("")),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if tc.expPanics == nil {
				merklePath := types.BuildMerklePath(tc.prefix, tc.path)
				s.Require().Equal(tc.expPath, merklePath)
			} else {
				s.Require().PanicsWithValue(*tc.expPanics, func() {
					_ = types.BuildMerklePath(tc.prefix, tc.path)
				})
			}
		})
	}
}
