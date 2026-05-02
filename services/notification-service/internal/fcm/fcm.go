package fcm

import (
	"context"
	"fmt"

	fb "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// Sender sends push notifications via Firebase Cloud Messaging.
type Sender interface {
	// Send sends a notification to a single device token.
	Send(ctx context.Context, msg *Message) (string, error)
	// SendEachForMulticast sends the same notification to multiple device tokens.
	// Returns per-token results so callers can detect invalid tokens.
	SendEachForMulticast(ctx context.Context, msg *MulticastMessage) (*BatchResponse, error)
}

// Message is a notification destined for a single device.
type Message struct {
	Token    string
	Title    string
	Body     string
	Data     map[string]string
	ImageURL string
}

// MulticastMessage is a notification destined for multiple devices.
type MulticastMessage struct {
	Tokens   []string
	Title    string
	Body     string
	Data     map[string]string
	ImageURL string
}

// SendResult holds the outcome for a single token within a multicast send.
type SendResult struct {
	Success bool
	Error   error
}

// BatchResponse aggregates the results of a multicast send.
type BatchResponse struct {
	SuccessCount int
	FailureCount int
	Results      []SendResult
}

type fcmSender struct {
	client *messaging.Client
}

// NewSender initialises a Firebase app and returns an FCM Sender.
func NewSender(ctx context.Context, credentialsFile, projectID string) (Sender, error) {
	var opts []option.ClientOption
	if credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}

	cfg := &fb.Config{}
	if projectID != "" {
		cfg.ProjectID = projectID
	}

	app, err := fb.NewApp(ctx, cfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to initialise firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("fcm: failed to get messaging client: %w", err)
	}

	return &fcmSender{client: client}, nil
}

func (s *fcmSender) Send(ctx context.Context, msg *Message) (string, error) {
	m := &messaging.Message{
		Token: msg.Token,
		Notification: &messaging.Notification{
			Title:    msg.Title,
			Body:     msg.Body,
			ImageURL: msg.ImageURL,
		},
		Data: msg.Data,
	}

	resp, err := s.client.Send(ctx, m)
	if err != nil {
		return "", fmt.Errorf("fcm: send failed: %w", err)
	}
	return resp, nil
}

func (s *fcmSender) SendEachForMulticast(ctx context.Context, msg *MulticastMessage) (*BatchResponse, error) {
	m := &messaging.MulticastMessage{
		Tokens: msg.Tokens,
		Notification: &messaging.Notification{
			Title:    msg.Title,
			Body:     msg.Body,
			ImageURL: msg.ImageURL,
		},
		Data: msg.Data,
	}

	resp, err := s.client.SendEachForMulticast(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("fcm: multicast send failed: %w", err)
	}

	br := &BatchResponse{
		SuccessCount: resp.SuccessCount,
		FailureCount: resp.FailureCount,
		Results:      make([]SendResult, len(resp.Responses)),
	}
	for i, r := range resp.Responses {
		br.Results[i] = SendResult{
			Success: r.Success,
			Error:   r.Error,
		}
	}

	return br, nil
}