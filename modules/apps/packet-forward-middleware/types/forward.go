package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/iancoleman/orderedmap"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

type PacketMetadata struct {
	Forward *ForwardMetadata `json:"forward"`
}

type ForwardMetadata struct {
	Receiver string   `json:"receiver,omitempty"`
	Port     string   `json:"port,omitempty"`
	Channel  string   `json:"channel,omitempty"`
	Timeout  Duration `json:"timeout,omitempty"`
	Retries  *uint8   `json:"retries,omitempty"`

	// Using JSONObject so that objects for next property will not be mutated by golang's lexicographic key sort on map keys during Marshal.
	// Supports primitives for Unmarshal/Marshal so that an escaped JSON-marshaled string is also valid.
	Next *JSONObject `json:"next,omitempty"`
}

type Duration time.Duration

func (m *ForwardMetadata) Validate() error {
	if m.Receiver == "" {
		return fmt.Errorf("failed to validate metadata. receiver cannot be empty")
	}
	if err := host.PortIdentifierValidator(m.Port); err != nil {
		return fmt.Errorf("failed to validate metadata: %w", err)
	}
	if err := host.ChannelIdentifierValidator(m.Channel); err != nil {
		return fmt.Errorf("failed to validate metadata: %w", err)
	}

	return nil
}

// JSONObject is a wrapper type to allow either a primitive type or a JSON object.
// In the case the value is a JSON object, OrderedMap type is used so that key order
// is retained across Unmarshal/Marshal.
type JSONObject struct {
	obj        bool
	primitive  []byte
	orderedMap orderedmap.OrderedMap
}

// NewJSONObject is a constructor used for tests.
// The usage of JSONObject in the middleware is only json Marshal/Unmarshal
func NewJSONObject(object bool, primitive []byte, orderedMap orderedmap.OrderedMap) *JSONObject {
	return &JSONObject{
		obj:        object,
		primitive:  primitive,
		orderedMap: orderedMap,
	}
}

// UnmarshalJSON overrides the default json.Unmarshal behavior
func (o *JSONObject) UnmarshalJSON(b []byte) error {
	if err := o.orderedMap.UnmarshalJSON(b); err != nil {
		// If ordered map unmarshal fails, this is a primitive value
		o.obj = false
		// Attempt to unmarshal as string, this removes extra JSON escaping
		var primitiveStr string
		if err := json.Unmarshal(b, &primitiveStr); err != nil {
			o.primitive = b
			return nil
		}
		o.primitive = []byte(primitiveStr)
		return nil
	}
	// This is a JSON object, now stored as an ordered map to retain key order.
	o.obj = true
	return nil
}

// MarshalJSON overrides the default json.Marshal behavior
func (o JSONObject) MarshalJSON() ([]byte, error) {
	if o.obj {
		// non-primitive, return marshaled ordered map.
		return o.orderedMap.MarshalJSON()
	}
	// primitive, return raw bytes.
	return o.primitive, nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).Nanoseconds())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}
