package types_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
)

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
	tests := []struct {
		name      string
		metadata  types.ForwardMetadata
		expectErr bool
	}{
		{
			name: "valid metadata",
			metadata: types.ForwardMetadata{
				Receiver: "validaddress",
				Port:     "validport",
				Channel:  "validchannel",
				Timeout:  time.Duration(0),
				Retries:  nil,
			},
			expectErr: false,
		},
		{
			name: "empty receiver",
			metadata: types.ForwardMetadata{
				Receiver: "",
				Port:     "validport",
				Channel:  "validchannel",
				Timeout:  time.Duration(0),
				Retries:  nil,
			},
			expectErr: true,
		},
		{
			name: "invalid port",
			metadata: types.ForwardMetadata{
				Receiver: "validaddress",
				Port:     "!nv@lidport",
				Channel:  "validchannel",
				Timeout:  time.Duration(0),
				Retries:  nil,
			},
			expectErr: true,
		},
		{
			name: "invalid channel",
			metadata: types.ForwardMetadata{
				Receiver: "validaddress",
				Port:     "validport",
				Channel:  "invalid|channel",
				Timeout:  time.Duration(0),
				Retries:  nil,
			},
			expectErr: true,
		},
		{
			name: "valid metadata with next",
			metadata: types.ForwardMetadata{
				Receiver: "validaddress",
				Port:     "validport",
				Channel:  "validchannel",
				Timeout:  time.Duration(0),
				Retries:  nil,
				Next: &types.PacketMetadata{
					Forward: types.ForwardMetadata{
						Receiver: "nextreceiver",
						Port:     "nextport",
						Channel:  "nextchannel",
					},
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestForwardMetadataToMap(t *testing.T) {
	metadata := types.ForwardMetadata{
		Receiver: "receiver",
		Port:     "port",
		Channel:  "channel",
		Timeout:  time.Duration(30),
		Retries:  func(v uint8) *uint8 { return &v }(2),
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
			Port:     "nextport",
			Channel:  "nextchannel",
		},
	}

	m = metadata.ToMap()
	next, ok := m["next"].(map[string]any)["forward"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "nextreceiver", next["receiver"])
}

func TestPacketMetadataToMemo(t *testing.T) {
	metadata := types.PacketMetadata{
		Forward: types.ForwardMetadata{
			Receiver: "receiver",
			Port:     "port",
			Channel:  "channel",
			Timeout:  time.Duration(30),
			Retries:  func(v uint8) *uint8 { return &v }(2),
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
	tests := []struct {
		name               string
		customData         map[string]any
		expectedError      string
		expectedHasForward bool
	}{
		{
			name: "invalid timeout string",
			customData: map[string]any{
				"forward": map[string]any{
					"receiver": "test-receiver",
					"port":     "test-port",
					"channel":  "test-channel",
					"timeout":  "invalid-duration",
				},
			},
			expectedError:      "time: invalid duration",
			expectedHasForward: true,
		},
		{
			name: "retries value too high",
			customData: map[string]any{
				"forward": map[string]any{
					"receiver": "test-receiver",
					"port":     "test-port",
					"channel":  "test-channel",
					"retries":  float64(300), // > 255
				},
			},
			expectedError:      "retries must be between 0 and 255",
			expectedHasForward: true,
		},
		{
			name: "invalid retries type",
			customData: map[string]any{
				"forward": map[string]any{
					"receiver": "test-receiver",
					"port":     "test-port",
					"channel":  "test-channel",
					"retries":  "not-a-number", // Invalid type
				},
			},
			expectedError:      "key retries has invalid type, expected number",
			expectedHasForward: true,
		},
		{
			name: "invalid next JSON string",
			customData: map[string]any{
				"forward": map[string]any{
					"receiver": "test-receiver",
					"port":     "test-port",
					"channel":  "test-channel",
					"next":     "invalid json",
				},
			},
			expectedError:      "failed to unmarshal next forward metadata",
			expectedHasForward: true,
		},
		{
			name: "missing receiver",
			customData: map[string]any{
				"forward": map[string]any{
					"port":    "test-port",
					"channel": "test-channel",
				},
			},
			expectedError:      "receiver",
			expectedHasForward: true,
		},
		{
			name: "missing port",
			customData: map[string]any{
				"forward": map[string]any{
					"receiver": "test-receiver",
					"channel":  "test-channel",
				},
			},
			expectedError:      "port",
			expectedHasForward: true,
		},
		{
			name: "missing channel",
			customData: map[string]any{
				"forward": map[string]any{
					"receiver": "test-receiver",
					"port":     "test-port",
				},
			},
			expectedError:      "channel",
			expectedHasForward: true,
		},
		{
			name: "nested forward metadata error",
			customData: map[string]any{
				"forward": map[string]any{
					"receiver": "test-receiver",
					"port":     "test-port",
					"channel":  "test-channel",
					"next": map[string]any{
						"forward": map[string]any{
							"receiver": "nested-receiver",
							"port":     "nested-port",
							// Missing required "channel" key
						},
					},
				},
			},
			expectedError:      "failed to get next forward metadata from packet data",
			expectedHasForward: true,
		},
		{
			name: "invalid next type",
			customData: map[string]any{
				"forward": map[string]any{
					"receiver": "test-receiver",
					"port":     "test-port",
					"channel":  "test-channel",
					"next":     42, // Invalid type (not map or string)
				},
			},
			expectedError:      "next forward metadata is not a valid map or string",
			expectedHasForward: true,
		},
		{
			name: "missing forward key in next metadata",
			customData: map[string]any{
				"forward": map[string]any{
					"receiver": "test-receiver",
					"port":     "test-port",
					"channel":  "test-channel",
					"next": map[string]any{
						"other_key": "some_value", // Missing "forward" key
					},
				},
			},
			expectedError:      "key forward not found in next forward metadata",
			expectedHasForward: true,
		},
		{
			name: "invalid timeout type",
			customData: map[string]any{
				"forward": map[string]any{
					"receiver": "test-receiver",
					"port":     "test-port",
					"channel":  "test-channel",
					"timeout":  true, // Invalid type (boolean instead of duration)
				},
			},
			expectedError:      "invalid duration",
			expectedHasForward: true,
		},
		{
			name:               "missing forward key entirely",
			customData:         map[string]any{},
			expectedError:      "key forward not found in packet data",
			expectedHasForward: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &MockPacketDataProvider{
				customData: tt.customData,
			}

			_, hasForward, err := types.GetPacketMetadataFromPacketdata(mockProvider)
			require.Error(t, err)
			require.Equal(t, tt.expectedHasForward, hasForward)
			require.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
