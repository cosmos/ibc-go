package types_test

import (
	"encoding/json"
	"testing"

	"github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
	"github.com/stretchr/testify/require"
)

func TestForwardMetadataUnmarshalStringNext(t *testing.T) {
	const memo = "{\"forward\":{\"receiver\":\"noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8\",\"port\":\"transfer\",\"channel\":\"channel-0\",\"timeout\":0,\"next\":\"{\\\"forward\\\":{\\\"receiver\\\":\\\"noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp\\\",\\\"port\\\":\\\"transfer\\\",\\\"channel\\\":\\\"channel-0\\\",\\\"timeout\\\":0}}\"}}"
	var packetMetadata types.PacketMetadata

	err := json.Unmarshal([]byte(memo), &packetMetadata)
	require.NoError(t, err)

	nextBz, err := json.Marshal(packetMetadata.Forward.Next)
	require.NoError(t, err)
	require.Equal(t, `{"forward":{"receiver":"noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp","port":"transfer","channel":"channel-0","timeout":0}}`, string(nextBz))
}

func TestForwardMetadataUnmarshalJSONNext(t *testing.T) {
	const memo = "{\"forward\":{\"receiver\":\"noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8\",\"port\":\"transfer\",\"channel\":\"channel-0\",\"timeout\":0,\"next\":{\"forward\":{\"receiver\":\"noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp\",\"port\":\"transfer\",\"channel\":\"channel-0\",\"timeout\":0}}}}"
	var packetMetadata types.PacketMetadata

	err := json.Unmarshal([]byte(memo), &packetMetadata)
	require.NoError(t, err)

	nextBz, err := json.Marshal(packetMetadata.Forward.Next)
	require.NoError(t, err)
	require.Equal(t, `{"forward":{"receiver":"noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp","port":"transfer","channel":"channel-0","timeout":0}}`, string(nextBz))
}

func TestTimeoutUnmarshalString(t *testing.T) {
	const memo = "{\"forward\":{\"receiver\":\"noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8\",\"port\":\"transfer\",\"channel\":\"channel-0\",\"timeout\":\"60s\"}}"
	var packetMetadata types.PacketMetadata

	err := json.Unmarshal([]byte(memo), &packetMetadata)
	require.NoError(t, err)

	timeoutBz, err := json.Marshal(packetMetadata.Forward.Timeout)
	require.NoError(t, err)

	require.Equal(t, "60000000000", string(timeoutBz))
}

func TestTimeoutUnmarshalJSON(t *testing.T) {
	const memo = "{\"forward\":{\"receiver\":\"noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8\",\"port\":\"transfer\",\"channel\":\"channel-0\",\"timeout\": 60000000000}}"
	var packetMetadata types.PacketMetadata

	err := json.Unmarshal([]byte(memo), &packetMetadata)
	require.NoError(t, err)

	timeoutBz, err := json.Marshal(packetMetadata.Forward.Timeout)
	require.NoError(t, err)

	require.Equal(t, "60000000000", string(timeoutBz))
}
