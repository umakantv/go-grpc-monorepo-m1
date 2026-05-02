package notificationclient

import (
	"context"
	"time"

	notifv1 "github.com/yourorg/monorepo/gen/go/private/notification"
	"github.com/yourorg/monorepo/pkg/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the gRPC notification service client.
type Client struct {
	conn   *grpc.ClientConn
	client notifv1.NotificationServiceClient
}

// New creates a new notification service client.
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
		client: notifv1.NewNotificationServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// RegisterDeviceToken registers a device token for push notifications.
func (c *Client) RegisterDeviceToken(ctx context.Context, userID, token, platform, deviceName string) (*notifv1.RegisterDeviceTokenResponse, error) {
	return c.client.RegisterDeviceToken(ctx, &notifv1.RegisterDeviceTokenRequest{
		UserId:     userID,
		Token:      token,
		Platform:   platform,
		DeviceName: deviceName,
	})
}

// UnregisterDeviceToken deactivates a device token.
func (c *Client) UnregisterDeviceToken(ctx context.Context, token string) (*notifv1.UnregisterDeviceTokenResponse, error) {
	return c.client.UnregisterDeviceToken(ctx, &notifv1.UnregisterDeviceTokenRequest{
		Token: token,
	})
}

// ListDeviceTokens lists all device tokens for a user.
func (c *Client) ListDeviceTokens(ctx context.Context, userID string) (*notifv1.ListDeviceTokensResponse, error) {
	return c.client.ListDeviceTokens(ctx, &notifv1.ListDeviceTokensRequest{
		UserId: userID,
	})
}

// SendNotification sends a push notification to all devices of a user.
func (c *Client) SendNotification(ctx context.Context, userID, title, body string, data map[string]string, imageURL string) (*notifv1.SendNotificationResponse, error) {
	return c.client.SendNotification(ctx, &notifv1.SendNotificationRequest{
		UserId:   userID,
		Title:    title,
		Body:     body,
		Data:     data,
		ImageUrl: imageURL,
	})
}

// SendBulkNotification sends the same notification to multiple users.
func (c *Client) SendBulkNotification(ctx context.Context, userIDs []string, title, body string, data map[string]string, imageURL string) (*notifv1.SendBulkNotificationResponse, error) {
	return c.client.SendBulkNotification(ctx, &notifv1.SendBulkNotificationRequest{
		UserIds:  userIDs,
		Title:    title,
		Body:     body,
		Data:     data,
		ImageUrl: imageURL,
	})
}

// GetNotification retrieves a notification by ID.
func (c *Client) GetNotification(ctx context.Context, notificationID string) (*notifv1.GetNotificationResponse, error) {
	return c.client.GetNotification(ctx, &notifv1.GetNotificationRequest{
		NotificationId: notificationID,
	})
}

// ListNotifications lists notifications for a user.
func (c *Client) ListNotifications(ctx context.Context, userID string, pageSize int32, pageToken string) (*notifv1.ListNotificationsResponse, error) {
	return c.client.ListNotifications(ctx, &notifv1.ListNotificationsRequest{
		UserId:    userID,
		PageSize:  pageSize,
		PageToken: pageToken,
	})
}