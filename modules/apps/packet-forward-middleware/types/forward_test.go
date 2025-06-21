package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

func TestForwardMetadataUnmarshalStringNext(t *testing.T) {
	const memo = `{ "forward": {
		"receiver":"noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8",
		"port":"transfer",
		"channel":"channel-0",
		"timeout":0,
		"next": "{ \"forward\":{\"receiver\":\"noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp\",\"port\":\"transfer\",\"channel\":\"channel-0\",\"timeout\":0 } }"
	}}`

	packetData := transfertypes.InternalTransferRepresentation{
		Memo: memo,
	}

	packetMetadata, isPFM, err := types.GetPacketMetadataFromPacketdata(packetData)
	require.True(t, isPFM, "expected packet metadata to be a PFM packet")
	forwardMetadata := packetMetadata.Forward
	require.NoError(t, err)
	require.Equal(t, "noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8", forwardMetadata.Receiver)
	require.Equal(t, "transfer", forwardMetadata.Port)
	require.Equal(t, "channel-0", forwardMetadata.Channel)
	require.Equal(t, time.Duration(0), forwardMetadata.Timeout)
	require.Nil(t, forwardMetadata.Retries)
	require.NotNil(t, forwardMetadata.Next)

	nextForwardMetadata := forwardMetadata.Next.Forward
	require.Equal(t, "noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp", nextForwardMetadata.Receiver)
	require.Equal(t, "transfer", nextForwardMetadata.Port)
	require.Equal(t, "channel-0", nextForwardMetadata.Channel)
	require.Equal(t, time.Duration(0), nextForwardMetadata.Timeout)
	require.Nil(t, nextForwardMetadata.Retries)
	require.Nil(t, nextForwardMetadata.Next)
}

func TestForwardMetadataUnmarshalJSONNext(t *testing.T) {
	const memo = `{ "forward": {
		"receiver":"noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8",
		"port":"transfer",
		"channel":"channel-0",
		"timeout":0,
		"next": { "forward": {
			"receiver":"noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp",
			"port":"transfer",
			"channel":"channel-0",
			"timeout":0
		}}
	}}`

	packetData := transfertypes.InternalTransferRepresentation{
		Memo: memo,
	}

	packetMetadata, isPFM, err := types.GetPacketMetadataFromPacketdata(packetData)
	require.True(t, isPFM, "expected packet metadata to be a PFM packet")
	forwardMetadata := packetMetadata.Forward
	require.NoError(t, err)
	require.Equal(t, "noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8", forwardMetadata.Receiver)
	require.Equal(t, "transfer", forwardMetadata.Port)
	require.Equal(t, "channel-0", forwardMetadata.Channel)
	require.Equal(t, time.Duration(0), forwardMetadata.Timeout)
	require.Nil(t, forwardMetadata.Retries)
	require.NotNil(t, forwardMetadata.Next)

	nextForwardMetadata := forwardMetadata.Next.Forward
	require.Equal(t, "noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp", nextForwardMetadata.Receiver)
	require.Equal(t, "transfer", nextForwardMetadata.Port)
	require.Equal(t, "channel-0", nextForwardMetadata.Channel)
	require.Equal(t, time.Duration(0), nextForwardMetadata.Timeout)
	require.Nil(t, nextForwardMetadata.Retries)
	require.Nil(t, nextForwardMetadata.Next)
}

func TestTimeoutUnmarshalString(t *testing.T) {
	const memo = `{ "forward": {
		"receiver":"noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8",
		"port":"transfer",
		"channel":"channel-0",
		"timeout":"60s"
	}}`

	packetData := transfertypes.InternalTransferRepresentation{
		Memo: memo,
	}

	packetMetadata, isPFM, err := types.GetPacketMetadataFromPacketdata(packetData)
	require.True(t, isPFM, "expected packet metadata to be a PFM packet")
	forwardMetadata := packetMetadata.Forward
	require.NoError(t, err)
	require.Equal(t, "noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8", forwardMetadata.Receiver)
	require.Equal(t, "transfer", forwardMetadata.Port)
	require.Equal(t, "channel-0", forwardMetadata.Channel)
	require.Equal(t, 60*time.Second, forwardMetadata.Timeout)
	require.Nil(t, forwardMetadata.Retries)
	require.Nil(t, forwardMetadata.Next)
}

func TestTimeoutUnmarshalJSON(t *testing.T) {
	const memo = `{ "forward": {
		"receiver":"noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8",
		"port":"transfer",
		"channel":"channel-0",
		"timeout": 60000000000
	}}`

	packetData := transfertypes.InternalTransferRepresentation{
		Memo: memo,
	}

	packetMetadata, isPFM, err := types.GetPacketMetadataFromPacketdata(packetData)
	require.True(t, isPFM, "expected packet metadata to be a PFM packet")
	forwardMetadata := packetMetadata.Forward
	require.NoError(t, err)
	require.Equal(t, "noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8", forwardMetadata.Receiver)
	require.Equal(t, "transfer", forwardMetadata.Port)
	require.Equal(t, "channel-0", forwardMetadata.Channel)
	require.Equal(t, 60*time.Second, forwardMetadata.Timeout)
	require.Nil(t, forwardMetadata.Retries)
	require.Nil(t, forwardMetadata.Next)
}

func TestEmptyMemo(t *testing.T) {
	packetData := transfertypes.InternalTransferRepresentation{
		Memo: "",
	}

	_, isPFM, _ := types.GetPacketMetadataFromPacketdata(packetData)
	require.False(t, isPFM, "expected packet metadata to not be a PFM packet")
}

func TestInvalidMetadata(t *testing.T) {
	const memo = `{ "forward": {
		"receiver":"noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8",
		"port":"transfer",
		"channel":"channel-0",
		"timeout": "invalid"
	}}`

	packetData := transfertypes.InternalTransferRepresentation{
		Memo: memo,
	}

	packetMetadata, isPFM, err := types.GetPacketMetadataFromPacketdata(packetData)
	require.True(t, isPFM, "expected packet metadata to be a PFM packet")
	require.Error(t, err, "expected error due to invalid timeout format")
	require.Nil(t, packetMetadata.Forward.Next)
}
