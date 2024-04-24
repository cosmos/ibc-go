package celestia_test

import (
	"encoding/hex"
	"encoding/json"

	celestia "github.com/cosmos/ibc-go/modules/light-clients/07-celestia"
	"github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// dataHash queried from commit at height=10000
// https://public-celestia-rpc.numia.xyz/header?height=10000
const dataHash string = "694F52677DDA82F3148D0A170ECC2A6A74A72563CC3F042BA7277AF3C1558127"

// shareProofJSON for shares [0,1] queried at height=10000
// https://public-celestia-rpc.numia.xyz/prove_shares?height=10000&startShare=0&endShare=1
var shareProofJSON = `{
    "data": [
      "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEBAAAENAAAACbsAgrAAQq9AQopL2liYy5hcHBsaWNhdGlvbnMudHJhbnNmZXIudjEuTXNnVHJhbnNmZXISjwEKCHRyYW5zZmVyEgljaGFubmVsLTIaEQoEdXRpYRIJMjU3MDAwMDAwIi9jZWxlc3RpYTF3ZWU2MjRscG53Y2NkcHB5dDJrZGZ5czZydnI5eHl2dThmbWVhZSorb3NtbzF3ZWU2MjRscG53Y2NkcHB5dDJrZGZ5czZydnI5eHl2dTdjZWUzeDIHCAEQ+p3mBRJlCk4KRgofL2Nvc21vcy5jcnlwdG8uc2VjcDI1NmsxLlB1YktleRIjCiEDQTeSPV//mL4658YJPa9fwwXHbo0zygD9Q0FWgyFN/QwSBAoCCH8SEwoNCgR1dGlhEgUyNzYyORCftwgaQBQg6ZFAoqJz3duveBY3NuDk7llRaGu/e6PQPTaoz4ZGNW5qZkFjgYhkglPXMEdKdktaNO/VEU+RZztQcIwE49DRAgqlAQqiAQojL2Nvc21vcy5zdGFraW5nLnYxYmV0YTEuTXNnRGVsZWdhdGUSewovY2VsZXN0aWExemVnN2xycXNjNjh5NW1nZHBnbnp5ZGUza3E3eDJ0cnE3NnZraDkSNmNlbGVzdGlhdmE="
    ],
    "share_proofs": [
      {
        "end": 1,
        "nodes": [
          "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAVCj8zV410D/2gZY+0iDX7AS/snewF0EQGkuuPWq24Te",
          "/////////////////////////////////////////////////////////////////////////////7kDYk5RAaznnwDfIPmd6Iyw03FvBjhlOrAQEMY6dnNd"
        ]
      }
    ],
    "namespace_id": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQ==",
    "row_proof": {
      "row_roots": [
        "00000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000001D3C808F4BA0439B72D803F35FC92ED42B86A12C1121CD3A72EC07BE00FAF0DAD"
      ],
      "proofs": [
        {
          "total": 8,
          "index": 0,
          "leaf_hash": "Oj+WbIzd1NCo4e4ptcPkyCR3vRW1eI28Fu+GfvCi1hk=",
          "aunts": [
            "Dm+pi7IQIqqeDq5sA6aDRl29AddOikyIMsNKTs3TOvQ=",
            "EN4RBE6ZgyzZAOYbzCPaNgUuxZv1F9a3Av0oqQ7VPEo=",
            "ibs0Ape4CNv+qCosdw8W/m4ADHIt6HyqLRVMyNF5FqE="
          ]
        }
      ],
      "start_row": 0,
      "end_row": 0
    },
    "namespace_version": 0
  }`

func (suite *CelestiaTestSuite) TestVerifyMembership() {
	var (
		height exported.Height
		path   *ibctesting.Path
		proof  []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: with share proof queried from celestia-core",
			func() {
				// convert ShareProof json to protobuf encoded bytes
				var shareProofInternal celestia.ShareProofInternal
				err := json.Unmarshal([]byte(shareProofJSON), &shareProofInternal)
				suite.Require().NoError(err)

				shareProofProto := shareProofInternal.ToProto()

				bz, err := suite.chainA.App.AppCodec().Marshal(&shareProofProto)
				suite.Require().NoError(err)

				proof = bz // assign proof bytes to ShareProof proto

				// overwrite the client consensus state data root to the stub dataHash from queried Height
				consensusState := path.EndpointA.GetConsensusState(height)
				tmConsensusState, ok := consensusState.(*ibctm.ConsensusState)
				suite.Require().True(ok)

				root, err := hex.DecodeString(dataHash)
				suite.Require().NoError(err)

				// assign consensus state root as data availability root
				tmConsensusState.Root = types.NewMerkleRoot(root)

				path.EndpointA.SetConsensusState(tmConsensusState, height)
			},
			nil,
		},
		// TODO: query blob.Proof from celestia-node API and plug in here
		// {
		// 	"failure: with proofs from celestia-node blob.GetProof api",
		// 	func() {
		// 	},
		// 	nil,
		// },
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			path = ibctesting.NewPath(suite.chainA, suite.chainB)

			clientID := suite.CreateClient(path.EndpointA)

			lightClientModule, found := suite.chainA.App.GetIBCKeeper().ClientKeeper.Route(clientID)
			suite.Require().True(found)

			height = path.EndpointA.GetClientLatestHeight()

			tc.malleate()

			err := lightClientModule.VerifyMembership(suite.chainA.GetContext(), clientID, height, 0, 0, proof, nil, nil)

			expPass := tc.expError == nil
			if expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().ErrorIs(err, tc.expError, "failed verify membership")
			}
		})
	}
}
