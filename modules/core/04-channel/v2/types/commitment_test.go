package types_test

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/cosmos/solidity-ibc-eureka/abigen/ics20lib"
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
			"a096722aa6534040a0efbdae05765132a7b223ad306d6512f3734821bd046505",
		},
		{
			"abi packet",
			func() {
				transferData, err := ics20lib.EncodeFungibleTokenPacketData(ics20lib.IICS20TransferMsgsFungibleTokenPacketDataV2{
					Tokens: []ics20lib.IICS20TransferMsgsToken{
						{
							Denom: ics20lib.IICS20TransferMsgsDenom{
								Base: "uatom",
								Trace: []ics20lib.IICS20TransferMsgsHop{
									{
										PortId:   "traceport",
										ClientId: "client-0",
									},
								},
							},
							Amount: big.NewInt(1_000_000),
						},
					},
					Sender:   "sender",
					Receiver: "receiver",
					Memo:     "memo",
					Forwarding: ics20lib.IICS20TransferMsgsForwardingPacketData{
						DestinationMemo: "destination-memo",
						Hops: []ics20lib.IICS20TransferMsgsHop{
							{
								PortId:   "hopport",
								ClientId: "client-1",
							},
						},
					},
				})
				require.NoError(t, err)
				packet.Payloads[0].Value = transferData
				packet.Payloads[0].Encoding = transfertypes.EncodingABI
				packet.Payloads[0].Version = transfertypes.V2
			},
			"634d50b132aadb0395ceb840bb613191326b5fc47248fd50e9e5c622ca11b59f",
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
}
