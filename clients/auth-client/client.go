package authclient

import (
	"context"
	"time"

	auth "github.com/yourorg/monorepo/gen/go/private/auth"
	"github.com/yourorg/monorepo/pkg/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the gRPC auth service client
type Client struct {
	conn   *grpc.ClientConn
	client auth.AuthServiceClient
}

// New creates a new auth service client with trace propagation enabled
func New(addr string) (*Client, error) {
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
		grpc.WithUnaryInterceptor(middleware.PropagationInterceptor()),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: auth.NewAuthServiceClient(conn),
	}, nil
}

// NewWithConn creates a new auth service client with an existing connection
func NewWithConn(conn *grpc.ClientConn) *Client {
	return &Client{
		conn:   conn,
		client: auth.NewAuthServiceClient(conn),
	}
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// ValidateToken validates a JWT token and returns user info
func (c *Client) ValidateToken(ctx context.Context, token string) (*auth.ValidateTokenResponse, error) {
	return c.client.ValidateToken(ctx, &auth.ValidateTokenRequest{
		Token: token,
	})
}

// GenerateToken generates a new JWT token for a user
func (c *Client) GenerateToken(ctx context.Context, userID, email string, roles []string, expiresIn int32) (*auth.GenerateTokenResponse, error) {
	return c.client.GenerateToken(ctx, &auth.GenerateTokenRequest{
		UserId:          userID,
		Email:           email,
		Roles:           roles,
		ExpiresInSeconds: expiresIn,
	})
}

// RefreshToken refreshes an existing token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*auth.RefreshTokenResponse, error) {
	return c.client.RefreshToken(ctx, &auth.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
}

// RevokeToken revokes a token (for logout)
func (c *Client) RevokeToken(ctx context.Context, token string) (*auth.RevokeTokenResponse, error) {
	return c.client.RevokeToken(ctx, &auth.RevokeTokenRequest{
		Token: token,
	})
}

// IsHealthy checks if the auth service is healthy
func (c *Client) IsHealthy(ctx context.Context) error {
	// Simple connectivity check - try to validate an empty token
	// The service should respond even if the token is invalid
	_, err := c.client.ValidateToken(ctx, &auth.ValidateTokenRequest{Token: ""})
	return err
}
