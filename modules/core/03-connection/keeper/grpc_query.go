package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
)

var _ types.QueryServer = (*queryServer)(nil)

// queryServer implements the 03-connection types.QueryServer interface.
// It embeds the connection keeper to leverage store access while limiting the api of the connection keeper.
type queryServer struct {
	*Keeper
}

// NewQueryServer returns a new 03-connection types.QueryServer implementation.
func NewQueryServer(k *Keeper) types.QueryServer {
	return &queryServer{
		Keeper: k,
	}
}

// Connection implements the Query/Connection gRPC method
func (q *queryServer) Connection(c context.Context, req *types.QueryConnectionRequest) (*types.QueryConnectionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ConnectionIdentifierValidator(req.ConnectionId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	connection, found := q.GetConnection(ctx, req.ConnectionId)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrap(types.ErrConnectionNotFound, req.ConnectionId).Error(),
		)
	}

	return &types.QueryConnectionResponse{
		Connection:  &connection,
		ProofHeight: clienttypes.GetSelfHeight(ctx),
	}, nil
}

// Connections implements the Query/Connections gRPC method
func (q *queryServer) Connections(c context.Context, req *types.QueryConnectionsRequest) (*types.QueryConnectionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	var connections []*types.IdentifiedConnection
	store := prefix.NewStore(ctx.KVStore(q.storeKey), []byte(host.KeyConnectionPrefix))

	pageRes, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		var result types.ConnectionEnd
		if err := q.cdc.Unmarshal(value, &result); err != nil {
			return err
		}

		connectionID, err := host.ParseConnectionPath(string(key))
		if err != nil {
			return err
		}

		identifiedConnection := types.NewIdentifiedConnection(connectionID, result)
		connections = append(connections, &identifiedConnection)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryConnectionsResponse{
		Connections: connections,
		Pagination:  pageRes,
		Height:      clienttypes.GetSelfHeight(ctx),
	}, nil
}

// ClientConnections implements the Query/ClientConnections gRPC method
func (q *queryServer) ClientConnections(c context.Context, req *types.QueryClientConnectionsRequest) (*types.QueryClientConnectionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ClientIdentifierValidator(req.ClientId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	clientConnectionPaths, found := q.GetClientConnectionPaths(ctx, req.ClientId)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrap(types.ErrClientConnectionPathsNotFound, req.ClientId).Error(),
		)
	}

	return &types.QueryClientConnectionsResponse{
		ConnectionPaths: clientConnectionPaths,
		ProofHeight:     clienttypes.GetSelfHeight(ctx),
	}, nil
}

// ConnectionClientState implements the Query/ConnectionClientState gRPC method
func (q *queryServer) ConnectionClientState(c context.Context, req *types.QueryConnectionClientStateRequest) (*types.QueryConnectionClientStateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ConnectionIdentifierValidator(req.ConnectionId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)

	connection, found := q.GetConnection(ctx, req.ConnectionId)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(types.ErrConnectionNotFound, "connection-id: %s", req.ConnectionId).Error(),
		)
	}

	clientState, found := q.clientKeeper.GetClientState(ctx, connection.ClientId)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(clienttypes.ErrClientNotFound, "client-id: %s", connection.ClientId).Error(),
		)
	}

	identifiedClientState := clienttypes.NewIdentifiedClientState(connection.ClientId, clientState)

	height := clienttypes.GetSelfHeight(ctx)
	return types.NewQueryConnectionClientStateResponse(identifiedClientState, nil, height), nil
}

// ConnectionConsensusState implements the Query/ConnectionConsensusState gRPC method
func (q *queryServer) ConnectionConsensusState(c context.Context, req *types.QueryConnectionConsensusStateRequest) (*types.QueryConnectionConsensusStateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ConnectionIdentifierValidator(req.ConnectionId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)

	connection, found := q.GetConnection(ctx, req.ConnectionId)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(types.ErrConnectionNotFound, "connection-id: %s", req.ConnectionId).Error(),
		)
	}

	height := clienttypes.NewHeight(req.RevisionNumber, req.RevisionHeight)
	consensusState, found := q.clientKeeper.GetClientConsensusState(ctx, connection.ClientId, height)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(clienttypes.ErrConsensusStateNotFound, "client-id: %s", connection.ClientId).Error(),
		)
	}

	anyConsensusState, err := clienttypes.PackConsensusState(consensusState)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	proofHeight := clienttypes.GetSelfHeight(ctx)
	return types.NewQueryConnectionConsensusStateResponse(connection.ClientId, anyConsensusState, height, nil, proofHeight), nil
}

// ConnectionParams implements the Query/ConnectionParams gRPC method.
func (q *queryServer) ConnectionParams(c context.Context, req *types.QueryConnectionParamsRequest) (*types.QueryConnectionParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := q.GetParams(ctx)

	return &types.QueryConnectionParamsResponse{
		Params: &params,
	}, nil
}
