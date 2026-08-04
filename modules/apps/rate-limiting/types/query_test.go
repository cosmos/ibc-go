package types_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
)

type recordingQueryClient struct {
	types.QueryClient
	method string
	id     string
	denom  string
}

func (c *recordingQueryClient) RateLimit(_ context.Context, req *types.QueryRateLimitRequest, _ ...grpc.CallOption) (*types.QueryRateLimitResponse, error) {
	c.method, c.id, c.denom = "RateLimit", req.ChannelOrClientId, req.Denom
	return &types.QueryRateLimitResponse{}, nil
}

func (c *recordingQueryClient) RateLimitsByChainID(_ context.Context, req *types.QueryRateLimitsByChainIDRequest, _ ...grpc.CallOption) (*types.QueryRateLimitsByChainIDResponse, error) {
	c.method, c.id = "RateLimitsByChainID", req.ChainId
	return &types.QueryRateLimitsByChainIDResponse{}, nil
}

func (c *recordingQueryClient) RateLimitsByChannelOrClientID(_ context.Context, req *types.QueryRateLimitsByChannelOrClientIDRequest, _ ...grpc.CallOption) (*types.QueryRateLimitsByChannelOrClientIDResponse, error) {
	c.method, c.id = "RateLimitsByChannelOrClientID", req.ChannelOrClientId
	return &types.QueryRateLimitsByChannelOrClientIDResponse{}, nil
}

func TestQueryRESTRoutes(t *testing.T) {
	testCases := []struct {
		method string
		path   string
		id     string
		denom  string
	}{
		{"RateLimitsByChainID", "/ibc/apps/rate-limiting/v1/chains/osmosis-1/ratelimits", "osmosis-1", ""},
		{"RateLimitsByChannelOrClientID", "/ibc/apps/rate-limiting/v1/clients/07-tendermint-0/ratelimits", "07-tendermint-0", ""},
		{"RateLimit", "/ibc/apps/rate-limiting/v1/clients/channel-0/ratelimits/denoms/transfer/channel-1/uatom", "channel-0", "transfer/channel-1/uatom"},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			client := &recordingQueryClient{}
			mux := runtime.NewServeMux()
			require.NoError(t, types.RegisterQueryHandlerClient(context.Background(), mux, client))

			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))

			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			require.Equal(t, tc.method, client.method)
			require.Equal(t, tc.id, client.id)
			require.Equal(t, tc.denom, client.denom)
		})
	}
}
