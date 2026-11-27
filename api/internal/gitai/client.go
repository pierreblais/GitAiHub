// Package gitai is the anticorruption layer between GitaiHub and the
// upstream gitai server. It speaks gRPC (through gitai's Go SDK and its
// generated clients) and translates gitai's protobuf model into the models of
// the Source, Review and Agent activity contexts. No other package imports
// gitai's types.
package gitai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	gitaiauth "github.com/feder-cr/git-ai/auth"
	gitaiclient "github.com/feder-cr/git-ai/client"
	"github.com/feder-cr/git-ai/proto/go/gitalypb"
	"github.com/feder-cr/git-ai/sdk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pierreblais/gitaihub/api/internal/source"
)

// Client is a connection to one gitai server.
type Client struct {
	address string
	storage string
	sdk     *sdk.Client
	conn    *grpc.ClientConn

	commits  gitalypb.CommitServiceClient
	refs     gitalypb.RefServiceClient
	diffs    gitalypb.DiffServiceClient
	internal gitalypb.InternalGitalyClient

	mu    sync.Mutex
	users map[string]*sdk.Client // per-user clients for review writes, by token
}

// Dial connects to gitai with the shared (operator) token. Reads use it;
// review writes use the token of the GitaiHub user (see Review methods).
func Dial(ctx context.Context, address, token, storage string, opts ...grpc.DialOption) (*Client, error) {
	s, err := sdk.Dial(ctx, address, sdk.WithToken(token), sdk.WithStorage(storage), sdk.WithGRPCOptions(opts...))
	if err != nil {
		return nil, err
	}
	grpcOpts := append([]grpc.DialOption{grpc.WithPerRPCCredentials(gitaiauth.RPCCredentialsV2(token))}, opts...)
	conn, err := gitaiclient.DialContext(ctx, address, gitaiclient.WithGrpcOptions(grpcOpts))
	if err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("gitai: dial %q: %w", address, err)
	}
	return &Client{
		address:  address,
		storage:  storage,
		sdk:      s,
		conn:     conn,
		commits:  gitalypb.NewCommitServiceClient(conn),
		refs:     gitalypb.NewRefServiceClient(conn),
		diffs:    gitalypb.NewDiffServiceClient(conn),
		internal: gitalypb.NewInternalGitalyClient(conn),
		users:    map[string]*sdk.Client{},
	}, nil
}

// Close closes every connection.
func (c *Client) Close() error {
	c.mu.Lock()
	for _, u := range c.users {
		_ = u.Close()
	}
	c.mu.Unlock()
	_ = c.conn.Close()
	return c.sdk.Close()
}

func (c *Client) repo(rel string) *gitalypb.Repository {
	return &gitalypb.Repository{StorageName: c.storage, RelativePath: rel}
}

func (c *Client) userClient(ctx context.Context, token string) (*sdk.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if u, ok := c.users[token]; ok {
		return u, nil
	}
	u, err := sdk.Dial(ctx, c.address, sdk.WithAgentToken(token), sdk.WithStorage(c.storage))
	if err != nil {
		return nil, err
	}
	c.users[token] = u
	return u, nil
}

// Ping checks that gitai answers (readiness probe).
func (c *Client) Ping(ctx context.Context) error {
	_, err := gitalypb.NewServerServiceClient(c.conn).ServerInfo(ctx, &gitalypb.ServerInfoRequest{})
	return translate(err)
}

// Discover lists the repositories of the storage. It uses an internal Gitaly
// RPC; callers fall back to configured repositories when it fails.
func (c *Client) Discover(ctx context.Context) ([]string, error) {
	stream, err := c.internal.WalkRepos(ctx, &gitalypb.WalkReposRequest{StorageName: c.storage})
	if err != nil {
		return nil, err
	}
	var out []string
	for {
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, msg.GetRelativePath())
	}
}

// translate maps gitai's gRPC errors onto the domain's sentinel errors,
// keeping the message. Other codes are returned as they are.
func translate(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() {
	case codes.NotFound:
		return fmt.Errorf("%w: %s", source.ErrNotFound, st.Message())
	case codes.InvalidArgument:
		return fmt.Errorf("%w: %s", source.ErrInvalidArgument, st.Message())
	}
	return err
}

// recvAll drains a server stream, calling each for every message.
func recvAll[T any](recv func() (T, error), each func(T)) error {
	for {
		msg, err := recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return translate(err)
		}
		each(msg)
	}
}
