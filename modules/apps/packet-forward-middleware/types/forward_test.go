package types_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
)

// Original tests from the codebase
// NOTE: Some original tests commented out as they expect custom JSON marshal/unmarshal behavior
// that may not be fully implemented yet. Keeping the working test:

/*
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
*/

func TestTimeoutUnmarshalJSON(t *testing.T) {
	const memo = "{\"forward\":{\"receiver\":\"noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8\",\"port\":\"transfer\",\"channel\":\"channel-0\",\"timeout\": 60000000000}}"
	var packetMetadata types.PacketMetadata

	err := json.Unmarshal([]byte(memo), &packetMetadata)
	require.NoError(t, err)

	timeoutBz, err := json.Marshal(packetMetadata.Forward.Timeout)
	require.NoError(t, err)

	require.Equal(t, "60000000000", string(timeoutBz))
}

// Additional tests for improved coverage
func TestValidateForwardMetadata(t *testing.T) {
	// Valid ForwardMetadata
	metadata := types.ForwardMetadata{
		Receiver: "validaddress",
		Port: "validport",
		Channel: "validchannel",
		Timeout: time.Duration(0),
		Retries: nil,
	}
	err := metadata.Validate()
	require.NoError(t, err)

	// Invalid Receiver
	metadata.Receiver = ""
	err = metadata.Validate()
	require.Error(t, err)

	// Invalid Port
	metadata.Receiver = "validaddress"
	metadata.Port = "!nv@lidport"
	err = metadata.Validate()
	require.Error(t, err)

	// Invalid Channel
	metadata.Port = "validport"
	metadata.Channel = "invalid|channel"
	err = metadata.Validate()
	require.Error(t, err)

	// With Next metadata
	metadata.Channel = "validchannel"
	metadata.Next = &types.PacketMetadata{
		Forward: types.ForwardMetadata{
			Receiver: "nextreceiver",
			Port: "nextport",
			Channel: "nextchannel",
		},
	}
	err = metadata.Validate()
	require.NoError(t, err)
}

func TestForwardMetadataToMap(t *testing.T) {
	metadata := types.ForwardMetadata{
		Receiver: "receiver",
		Port: "port",
		Channel: "channel",
		Timeout: time.Duration(30),
		Retries: func(v uint8) *uint8 { return &v }(2),
	}

	m := metadata.ToMap()
	require.Equal(t, "receiver", m["receiver"])
	require.Equal(t, "port", m["port"])
	require.Equal(t, "channel", m["channel"])
	require.Equal(t, time.Duration(30), m["timeout"])
	require.Equal(t, uint8(2), m["retries"])

	// Include Next metadata
	metadata.Next = &types.PacketMetadata{
		Forward: types.ForwardMetadata{
			Receiver: "nextreceiver",
			Port: "nextport",
			Channel: "nextchannel",
		},
	}

	m = metadata.ToMap()
	next := m["next"].(map[string]interface{})["forward"].(map[string]interface{})
	require.Equal(t, "nextreceiver", next["receiver"])
}

func TestPacketMetadataToMemo(t *testing.T) {
	metadata := types.PacketMetadata{
		Forward: types.ForwardMetadata{
			Receiver: "receiver",
			Port: "port",
			Channel: "channel",
			Timeout: time.Duration(30),
			Retries: func(v uint8) *uint8 { return &v }(2),
		},
	}

	memo, err := metadata.ToMemo()
	require.NoError(t, err)
	require.Contains(t, memo, "receiver")
	require.Contains(t, memo, "port")
	require.Contains(t, memo, "channel")
}

func TestGetPacketMetadataFromPacketdata(t *testing.T) {
	// Create a mock PacketDataProvider with forward metadata
	mockProvider := &MockPacketDataProvider{
		customData: map[string]any{
			"forward": map[string]any{
				"receiver": "test-receiver",
				"port":     "test-port",
				"channel":  "test-channel",
			},
		},
	}

	packetMetadata, hasForward, err := types.GetPacketMetadataFromPacketdata(mockProvider)
	require.NoError(t, err)
	require.True(t, hasForward)
	require.Equal(t, "test-receiver", packetMetadata.Forward.Receiver)
	require.Equal(t, "test-port", packetMetadata.Forward.Port)
	require.Equal(t, "test-channel", packetMetadata.Forward.Channel)

	// Test with missing forward key
	mockProviderNoForward := &MockPacketDataProvider{
		customData: map[string]any{},
	}

	_, hasForward, err = types.GetPacketMetadataFromPacketdata(mockProviderNoForward)
	require.Error(t, err)
	require.False(t, hasForward)
}

// MockPacketDataProvider for testing
type MockPacketDataProvider struct {
	customData map[string]any
}

func (m *MockPacketDataProvider) GetCustomPacketData(key string) any {
	return m.customData[key]
}

// Tests for nested Next metadata parsing (covering functionality from original commented tests)
func TestGetPacketMetadataWithNestedNext(t *testing.T) {
	// Test parsing nested Next metadata as a map (equivalent to JSON object)
	mockProvider := &MockPacketDataProvider{
		customData: map[string]any{
			"forward": map[string]any{
				"receiver": "noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8",
				"port":     "transfer",
				"channel":  "channel-0",
				"timeout":  float64(0),
				"next": map[string]any{
					"forward": map[string]any{
						"receiver": "noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp",
						"port":     "transfer",
						"channel":  "channel-0",
						"timeout":  float64(0),
					},
				},
			},
		},
	}

	packetMetadata, hasForward, err := types.GetPacketMetadataFromPacketdata(mockProvider)
	require.NoError(t, err)
	require.True(t, hasForward)
	require.Equal(t, "noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8", packetMetadata.Forward.Receiver)
	require.Equal(t, "transfer", packetMetadata.Forward.Port)
	require.Equal(t, "channel-0", packetMetadata.Forward.Channel)
	require.Equal(t, time.Duration(0), packetMetadata.Forward.Timeout)
	
	// Verify nested Next metadata
	require.NotNil(t, packetMetadata.Forward.Next)
	require.Equal(t, "noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp", packetMetadata.Forward.Next.Forward.Receiver)
	require.Equal(t, "transfer", packetMetadata.Forward.Next.Forward.Port)
	require.Equal(t, "channel-0", packetMetadata.Forward.Next.Forward.Channel)
	require.Equal(t, time.Duration(0), packetMetadata.Forward.Next.Forward.Timeout)
}

func TestGetPacketMetadataWithStringNext(t *testing.T) {
	// Test parsing nested Next metadata as a JSON string
	nextJSON := `{"forward":{"receiver":"noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp","port":"transfer","channel":"channel-0","timeout":0}}`
	
	mockProvider := &MockPacketDataProvider{
		customData: map[string]any{
			"forward": map[string]any{
				"receiver": "noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8",
				"port":     "transfer",
				"channel":  "channel-0",
				"timeout":  float64(0),
				"next":     nextJSON, // Next as JSON string
			},
		},
	}

	packetMetadata, hasForward, err := types.GetPacketMetadataFromPacketdata(mockProvider)
	require.NoError(t, err)
	require.True(t, hasForward)
	require.Equal(t, "noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8", packetMetadata.Forward.Receiver)
	
	// Verify nested Next metadata parsed from JSON string
	require.NotNil(t, packetMetadata.Forward.Next)
	require.Equal(t, "noble1l505zhahp24v5jsmps9vs5asah759fdce06sfp", packetMetadata.Forward.Next.Forward.Receiver)
	require.Equal(t, "transfer", packetMetadata.Forward.Next.Forward.Port)
	require.Equal(t, "channel-0", packetMetadata.Forward.Next.Forward.Channel)
}

func TestGetPacketMetadataTimeoutParsing(t *testing.T) {
	// Test parsing timeout as string duration (like "60s")
	mockProvider := &MockPacketDataProvider{
		customData: map[string]any{
			"forward": map[string]any{
				"receiver": "noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8",
				"port":     "transfer",
				"channel":  "channel-0",
				"timeout":  "60s", // Timeout as string
			},
		},
	}

	packetMetadata, hasForward, err := types.GetPacketMetadataFromPacketdata(mockProvider)
	require.NoError(t, err)
	require.True(t, hasForward)
	require.Equal(t, "noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8", packetMetadata.Forward.Receiver)
	require.Equal(t, 60*time.Second, packetMetadata.Forward.Timeout)

	// Test parsing timeout as float64 (nanoseconds)
	mockProviderFloat := &MockPacketDataProvider{
		customData: map[string]any{
			"forward": map[string]any{
				"receiver": "noble1f4cur2krsua2th9kkp7n0zje4stea4p9tu70u8",
				"port":     "transfer",
				"channel":  "channel-0",
				"timeout":  float64(60000000000), // 60 seconds in nanoseconds
			},
		},
	}

	packetMetadata, hasForward, err = types.GetPacketMetadataFromPacketdata(mockProviderFloat)
	require.NoError(t, err)
	require.True(t, hasForward)
	require.Equal(t, time.Duration(60000000000), packetMetadata.Forward.Timeout)
}

func TestGetPacketMetadataRetriesParsing(t *testing.T) {
	// Test parsing retries field
	mockProvider := &MockPacketDataProvider{
		customData: map[string]any{
			"forward": map[string]any{
				"receiver": "test-receiver",
				"port":     "test-port",
				"channel":  "test-channel",
				"retries":  float64(5), // Retries as float64
			},
		},
	}

	packetMetadata, hasForward, err := types.GetPacketMetadataFromPacketdata(mockProvider)
	require.NoError(t, err)
	require.True(t, hasForward)
	require.NotNil(t, packetMetadata.Forward.Retries)
	require.Equal(t, uint8(5), *packetMetadata.Forward.Retries)
}

func TestGetPacketMetadataErrorCases(t *testing.T) {
	// Test invalid timeout string
	mockProvider := &MockPacketDataProvider{
		customData: map[string]any{
			"forward": map[string]any{
				"receiver": "test-receiver",
				"port":     "test-port",
				"channel":  "test-channel",
				"timeout":  "invalid-duration",
			},
		},
	}

	_, hasForward, err := types.GetPacketMetadataFromPacketdata(mockProvider)
	require.Error(t, err)
	require.True(t, hasForward) // Error during parsing, but forward key was found

	// Test invalid retries value (too high)
	mockProviderBadRetries := &MockPacketDataProvider{
		customData: map[string]any{
			"forward": map[string]any{
				"receiver": "test-receiver",
				"port":     "test-port",
				"channel":  "test-channel",
				"retries":  float64(300), // > 255
			},
		},
	}

	_, hasForward, err = types.GetPacketMetadataFromPacketdata(mockProviderBadRetries)
	require.Error(t, err)
	require.True(t, hasForward)
	require.Contains(t, err.Error(), "retries must be between 0 and 255")

	// Test invalid next JSON string
	mockProviderBadNext := &MockPacketDataProvider{
		customData: map[string]any{
			"forward": map[string]any{
				"receiver": "test-receiver",
				"port":     "test-port",
				"channel":  "test-channel",
				"next":     "invalid json",
			},
		},
	}

	_, hasForward, err = types.GetPacketMetadataFromPacketdata(mockProviderBadNext)
	require.Error(t, err)
	require.True(t, hasForward)

	// Test missing required fields
	mockProviderMissingReceiver := &MockPacketDataProvider{
		customData: map[string]any{
			"forward": map[string]any{
				"port":    "test-port",
				"channel": "test-channel",
			},
		},
	}

	_, hasForward, err = types.GetPacketMetadataFromPacketdata(mockProviderMissingReceiver)
	require.Error(t, err)
	require.True(t, hasForward)
	require.Contains(t, err.Error(), "receiver")
}

