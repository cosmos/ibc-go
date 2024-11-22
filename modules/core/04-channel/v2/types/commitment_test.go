package types_test

import (
	"encoding/hex"
	"encoding/json"
	fmt "fmt"
	"testing"

	"github.com/stretchr/testify/require"

	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

// TestCommitPacket is primarily used to document the expected commitment output
// so that other implementations (such as the IBC Solidity) can replicate the
// same commitment output. But it is also useful to catch any changes in the commitment.
func TestCommitPacket(t *testing.T) {
	var packet types.Packet
	testCases := []struct {
		name         string
		malleate     func()
		expectedHash string
	}{
		{
			"json packet",
			func() {}, // default is json packet
			"450194f2ce25b12487f65593e106d91367a1e5c90b2efc03ed78265a54cfcebe",
		},
		{
			"abi packet",
			func() {
				transferData, err := transfertypes.EncodeABIFungibleTokenPacketData(transfertypes.FungibleTokenPacketData{
					Denom:    "uatom",
					Amount:   "1000000",
					Sender:   "sender",
					Receiver: "receiver",
					Memo:     "memo",
				})
				fmt.Printf("transferData: %s\n", string(transferData))
				fmt.Printf("hex value: %s\n", hex.EncodeToString(transferData))
				require.NoError(t, err)
				packet.Payloads[0].Value = transferData
				packet.Payloads[0].Encoding = "application/x-abi"
			},
			"5d47f1550f95d29d73c0e0af6940be5837cbd311b3c72d99fd6cb640081e7495",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transferData, err := json.Marshal(transfertypes.FungibleTokenPacketData{
				Denom:    "uatom",
				Amount:   "1000000",
				Sender:   "sender",
				Receiver: "receiver",
				Memo:     "memo",
			})
			require.NoError(t, err)
			packet = types.Packet{
				Sequence:           1,
				SourceChannel:      "channel-0",
				DestinationChannel: "channel-1",
				TimeoutTimestamp:   100,
				Payloads: []types.Payload{
					{
						SourcePort:      transfertypes.PortID,
						DestinationPort: transfertypes.PortID,
						Version:         transfertypes.V1,
						Encoding:        "application/json",
						Value:           transferData,
					},
				},
			}
			tc.malleate()
			commitment := types.CommitPacket(packet)
			require.Equal(t, tc.expectedHash, hex.EncodeToString(commitment))
			require.Len(t, commitment, 32)
		})
	}
}

// TestCommitAcknowledgement is primarily used to document the expected commitment output
// so that other implementations (such as the IBC Solidity) can replicate the
// same commitment output. But it is also useful to catch any changes in the commitment.
func TestCommitAcknowledgement(t *testing.T) {
	ack := types.Acknowledgement{
		AppAcknowledgements: [][]byte{
			[]byte("some bytes"),
		},
	}

	commitment := types.CommitAcknowledgement(ack)
	require.Equal(t, "f03b4667413e56aaf086663267913e525c442b56fa1af4fa3f3dab9f37044c5b", hex.EncodeToString(commitment))
}
