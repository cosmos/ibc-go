package collections_test

import (
	"testing"

	"github.com/cosmos/ibc-go/v7/internal/collections"
	"github.com/stretchr/testify/require"
)

func TestContainsString(t *testing.T) {
	testCases := []struct {
		name     string
		haystack []string
		needle   string
		expected bool
	}{
		{"failure empty haystack", []string{}, "needle", false},
		{"failure empty needle", []string{"hay", "stack"}, "", false},
		{"failure needle not in haystack", []string{"hay", "stack"}, "needle", false},
		{"success needle in haystack", []string{"hay", "stack", "needle"}, "needle", true},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expected, collections.Contains(tc.needle, tc.haystack), tc.name)
	}
}

func TestContainsInt(t *testing.T) {
	testCases := []struct {
		name     string
		haystack []int
		needle   int
		expected bool
	}{
		{"failure empty haystack", []int{}, 1, false},
		{"failure needle not in haystack", []int{1, 2, 3}, 4, false},
		{"success needle in haystack", []int{1, 2, 3, 4}, 4, true},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expected, collections.Contains(tc.needle, tc.haystack), tc.name)
	}
}
