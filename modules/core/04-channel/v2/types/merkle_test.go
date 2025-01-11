package types_test

import (
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v9/modules/core/23-commitment/types/v2"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (s *TypesTestSuite) TestBuildMerklePath() {
	path := ibctesting.NewPath(s.chainA, s.chainB)
	path.SetupV2()

	prefixPath := commitmenttypes.NewMerklePath([]byte("ibc"), []byte(""))
	packetCommitmentKey := host.PacketCommitmentKey(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, 1)

	testCases := []struct {
		name    string
		prefix  commitmenttypesv2.MerklePath
		path    []byte
		expPath commitmenttypesv2.MerklePath
	}{
		{
			name:    "standard ibc path",
			prefix:  prefixPath,
			path:    packetCommitmentKey,
			expPath: commitmenttypesv2.NewMerklePath([]byte("ibc"), packetCommitmentKey),
		},
		{
			name:    "non-empty last element prefix path",
			prefix:  commitmenttypes.NewMerklePath([]byte("ibc"), []byte("abc")),
			path:    packetCommitmentKey,
			expPath: commitmenttypesv2.NewMerklePath([]byte("ibc"), append([]byte("abc"), packetCommitmentKey...)),
		},
		{
			name:    "many elements in prefix path",
			prefix:  commitmenttypes.NewMerklePath([]byte("ibc"), []byte("a"), []byte("b"), []byte("c"), []byte("d")),
			path:    packetCommitmentKey,
			expPath: commitmenttypesv2.NewMerklePath([]byte("ibc"), []byte("a"), []byte("b"), []byte("c"), append([]byte("d"), packetCommitmentKey...)),
		},
		{
			name:    "empty prefix",
			prefix:  commitmenttypesv2.MerklePath{},
			path:    packetCommitmentKey,
			expPath: commitmenttypesv2.NewMerklePath(packetCommitmentKey),
		},
		{
			name:    "empty path",
			prefix:  prefixPath,
			path:    []byte{},
			expPath: commitmenttypesv2.NewMerklePath([]byte("ibc"), []byte("")),
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			merklePath := types.BuildMerklePath(tc.prefix, tc.path)
			s.Require().Equal(tc.expPath, merklePath)
		})
	}
}