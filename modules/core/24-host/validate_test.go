package host

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 195 characters
var longID = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis eros neque, ultricies vel ligula ac, convallis porttitor elit. Maecenas tincidunt turpis elit, vel faucibus nisl pellentesque sodales"

type testCase struct {
	msg    string
	id     string
	expErr error
}

func TestDefaultIdentifierValidator(t *testing.T) {
	testCases := []testCase{
		{"valid lowercase", "lowercaseid", nil},
		{"valid id special chars", "._+-#[]<>._+-#[]<>", nil},
		{"valid id lower and special chars", "lower._+-#[]<>", nil},
		{"numeric id", "1234567890", nil},
		{"uppercase id", "NOTLOWERCASE", nil},
		{"numeric id", "1234567890", nil},
		{"blank id", "               ", errors.New("the ID is blank - contains only spaces")},
		{"id length out of range", "1", errors.New("the ID length is too short")},
		{"id is too long", "this identifier is too long to be used as a valid identifier", errors.New("the ID exceeds the maximum allowed length")},
		{"path-like id", "lower/case/id", errors.New("the ID contains path-like characters, which are invalid for an ID")},
		{"invalid id", "(clientid)", errors.New("the ID contains invalid characters")},
		{"empty string", "", errors.New("the ID cannot be empty")},
	}

	for _, tc := range testCases {
		tc := tc

		err := ClientIdentifierValidator(tc.id)
		err1 := ConnectionIdentifierValidator(tc.id)
		err2 := ChannelIdentifierValidator(tc.id)
		err3 := PortIdentifierValidator(tc.id)
		if tc.expErr == nil {
			require.NoError(t, err, tc.msg)
			require.NoError(t, err1, tc.msg)
			require.NoError(t, err2, tc.msg)
			require.NoError(t, err3, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
			require.Error(t, err1, tc.msg)
			require.Error(t, err2, tc.msg)
			require.Error(t, err3, tc.msg)
		}
	}
}

func TestPortIdentifierValidator(t *testing.T) {
	testCases := []testCase{
		{"valid lowercase", "transfer", nil},
		{"valid id special chars", "._+-#[]<>._+-#[]<>", nil},
		{"valid id lower and special chars", "lower._+-#[]<>", nil},
		{"numeric id", "1234567890", nil},
		{"uppercase id", "NOTLOWERCASE", nil},
		{"numeric id", "1234567890", nil},
		{"blank id", "               ", errors.New("the ID is blank - contains only spaces")},
		{"id length out of range", "1", errors.New("the ID length is too short")},
		{"id is too long", longID, errors.New("the ID exceeds the maximum allowed length")},
		{"path-like id", "lower/case/id", errors.New("the ID contains path-like characters, which are invalid for an ID")},
		{"invalid id", "(clientid)", errors.New("the ID contains invalid characters")},
		{"empty string", "", errors.New("the ID cannot be empty")},
	}

	for _, tc := range testCases {
		tc := tc

		err := PortIdentifierValidator(tc.id)
		if tc.expErr == nil {
			require.NoError(t, err, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}

func TestPathValidator(t *testing.T) {
	testCases := []testCase{
		{"valid lowercase", "p/lowercaseid", nil},
		{"numeric path", "p/239123", nil},
		{"valid id special chars", "p/._+-#[]<>._+-#[]<>", nil},
		{"valid id lower and special chars", "lower/._+-#[]<>", nil},
		{"id length out of range", "p/l", nil},
		{"uppercase id", "p/NOTLOWERCASE", nil},
		{"invalid path", "lowercaseid", errors.New("the path is invalid")},
		{"blank id", "p/               ", errors.New("the ID is blank or contains only whitespace after the separator")},
		{"id length out of range", "p/12345678901234567890123456789012345678901234567890123456789012345", errors.New("the ID exceeds the maximum allowed length")},
		{"invalid id", "p/(clientid)", errors.New("the ID contains invalid characters, such as parentheses")},
		{"empty string", "", errors.New("the ID cannot be empty")},
		{"separators only", "////", errors.New("the ID contains only separators, which is invalid")},
		{"just separator", "/", errors.New("the ID cannot be just a separator")},
		{"begins with separator", "/id", errors.New("the ID should not begin with a separator")},
		{"blank before separator", "    /id", errors.New("the ID cannot have leading spaces before the separator")},
		{"ends with separator", "id/", errors.New("the ID cannot end with a separator")},
		{"blank after separator", "id/       ", errors.New("the ID cannot have trailing spaces after the separator")},
		{"blanks with separator", "  /  ", errors.New("the ID cannot have spaces before or after the separator")},
	}

	for _, tc := range testCases {
		tc := tc

		f := NewPathValidator(func(path string) error {
			return nil
		})

		err := f(tc.id)

		if tc.expErr == nil {
			seps := strings.Count(tc.id, "/")
			require.Equal(t, 1, seps)
			require.NoError(t, err, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}

func TestCustomPathValidator(t *testing.T) {
	validateFn := NewPathValidator(func(path string) error {
		if !strings.HasPrefix(path, "id_") {
			return fmt.Errorf("identifier %s must start with 'id_", path)
		}
		return nil
	})

	testCases := []testCase{
		{"valid custom path", "id_client/id_one", nil},
		{"invalid path", "client", errors.New("the path is invalid")},
		{"invalid custom path", "id_one/client", errors.New("the path contains an invalid structure")},
		{"invalid identifier", "id_client/id_1234567890123456789012345678901234567890123457890123456789012345", errors.New("the identifier exceeds the maximum allowed length for an ID")},
		{"separators only", "////", errors.New("the path contains only separators, which is invalid")},
		{"just separator", "/", errors.New("the path cannot be just a separator")},
		{"ends with separator", "id_client/id_one/", errors.New("the path cannot end with a separator")},
		{"beings with separator", "/id_client/id_one", errors.New("the path cannot begin with a separator")},
	}

	for _, tc := range testCases {
		tc := tc

		err := validateFn(tc.id)
		if tc.expErr == nil {
			require.NoError(t, err, tc.msg)
		} else {
			require.Error(t, err, tc.msg)
		}
	}
}
