package types_test

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
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
			"a096722aa6534040a0efbdae05765132a7b223ad306d6512f3734821bd046505",
		},
		{
			"abi packet",
			func() {
				transferData, err := transfertypes.EncodeABIFungibleTokenPacketData(&transfertypes.FungibleTokenPacketData{
					Denom:    "uatom",
					Amount:   "1000000",
					Sender:   "sender",
					Receiver: "receiver",
					Memo:     "memo",
				})
				require.NoError(t, err)
				packet.Payloads[0].Value = transferData
				packet.Payloads[0].Encoding = transfertypes.EncodingABI
			},
			"d408dca5088b9b375edb3c4df6bae0e18084fc0dbd90fcd0d028506553c81b25",
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
				Sequence:          1,
				SourceClient:      "07-tendermint-0",
				DestinationClient: "07-tendermint-1",
				TimeoutTimestamp:  100,
				Payloads: []types.Payload{
					{
						SourcePort:      transfertypes.PortID,
						DestinationPort: transfertypes.PortID,
						Version:         transfertypes.V1,
						Encoding:        transfertypes.EncodingJSON,
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

	failedAck := types.Acknowledgement{
		AppAcknowledgements: [][]byte{
			types.ErrorAcknowledgement[:],
		},
	}

	failedAckCommitment := types.CommitAcknowledgement(failedAck)
	require.Equal(t, "e2fb30dfbf7abdeaca82d426534d2b3a9d5444dd2a87fa16d38b77ba1a13ced7", hex.EncodeToString(failedAckCommitment))
}
