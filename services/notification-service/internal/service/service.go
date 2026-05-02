package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	notifv1 "github.com/yourorg/monorepo/gen/go/private/notification"
	"github.com/yourorg/monorepo/services/notification-service/internal/fcm"
	"github.com/yourorg/monorepo/services/notification-service/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NotificationService implements the NotificationService gRPC service.
type NotificationService struct {
	notifv1.UnimplementedNotificationServiceServer

	repo   *repository.Repository
	fcm    fcm.Sender
	logger *zap.Logger
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(repo *repository.Repository, sender fcm.Sender, logger *zap.Logger) *NotificationService {
	return &NotificationService{
		repo:   repo,
		fcm:    sender,
		logger: logger.Named("notification-service"),
	}
}

// ---- Device token RPCs ----

func (s *NotificationService) RegisterDeviceToken(ctx context.Context, req *notifv1.RegisterDeviceTokenRequest) (*notifv1.RegisterDeviceTokenResponse, error) {
	if req.GetUserId() == "" || req.GetToken() == "" || req.GetPlatform() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, token, and platform are required")
	}

	dt := &repository.DeviceToken{
		ID:         uuid.New().String(),
		UserID:     req.GetUserId(),
		Token:      req.GetToken(),
		Platform:   req.GetPlatform(),
		DeviceName: req.GetDeviceName(),
	}

	saved, err := s.repo.UpsertDeviceToken(ctx, dt)
	if err != nil {
		s.logger.Error("failed to register device token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to register device token")
	}

	s.logger.Info("device token registered",
		zap.String("user_id", saved.UserID),
		zap.String("platform", saved.Platform),
	)

	return &notifv1.RegisterDeviceTokenResponse{
		DeviceToken: deviceTokenToProto(saved),
	}, nil
}

func (s *NotificationService) UnregisterDeviceToken(ctx context.Context, req *notifv1.UnregisterDeviceTokenRequest) (*notifv1.UnregisterDeviceTokenResponse, error) {
	if req.GetToken() == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	err := s.repo.DeactivateDeviceToken(ctx, req.GetToken())
	if err != nil {
		if errors.Is(err, repository.ErrDeviceTokenNotFound) {
			return &notifv1.UnregisterDeviceTokenResponse{Success: true}, nil
		}
		s.logger.Error("failed to unregister device token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to unregister device token")
	}

	return &notifv1.UnregisterDeviceTokenResponse{Success: true}, nil
}

func (s *NotificationService) ListDeviceTokens(ctx context.Context, req *notifv1.ListDeviceTokensRequest) (*notifv1.ListDeviceTokensResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	tokens, err := s.repo.ListTokensByUserID(ctx, req.GetUserId())
	if err != nil {
		s.logger.Error("failed to list device tokens", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list device tokens")
	}

	protoTokens := make([]*notifv1.DeviceToken, len(tokens))
	for i, dt := range tokens {
		protoTokens[i] = deviceTokenToProto(dt)
	}

	return &notifv1.ListDeviceTokensResponse{DeviceTokens: protoTokens}, nil
}

// ---- Notification RPCs ----

func (s *NotificationService) SendNotification(ctx context.Context, req *notifv1.SendNotificationRequest) (*notifv1.SendNotificationResponse, error) {
	if req.GetUserId() == "" || req.GetTitle() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and title are required")
	}

	// Fetch active device tokens for the user.
	tokens, err := s.repo.ListActiveTokensByUserID(ctx, req.GetUserId())
	if err != nil {
		s.logger.Error("failed to fetch device tokens", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to send notification")
	}

	notifID := uuid.New().String()
	now := time.Now()

	if len(tokens) == 0 {
		// No devices registered; still log the notification.
		n := &repository.Notification{
			ID:       notifID,
			UserID:   req.GetUserId(),
			Title:    req.GetTitle(),
			Body:     req.GetBody(),
			Data:     req.GetData(),
			ImageURL: req.GetImageUrl(),
			Status:   repository.StatusSent,
			SentAt:   &now,
		}
		saved, _ := s.repo.CreateNotification(ctx, n)
		return &notifv1.SendNotificationResponse{
			Notification: notificationToProto(saved),
			DevicesSent:  0,
			DevicesFailed: 0,
		}, nil
	}

	// Build token list and send via FCM.
	tokenStrings := make([]string, len(tokens))
	for i, dt := range tokens {
		tokenStrings[i] = dt.Token
	}

	resp, err := s.fcm.SendEachForMulticast(ctx, &fcm.MulticastMessage{
		Tokens:   tokenStrings,
		Title:    req.GetTitle(),
		Body:     req.GetBody(),
		Data:     req.GetData(),
		ImageURL: req.GetImageUrl(),
	})

	notifStatus := repository.StatusSent
	var failureReason string
	if err != nil {
		notifStatus = repository.StatusFailed
		failureReason = err.Error()
		s.logger.Error("fcm multicast send failed", zap.Error(err))
	}

	var devicesSent, devicesFailed int32
	if resp != nil {
		devicesSent = int32(resp.SuccessCount)
		devicesFailed = int32(resp.FailureCount)

		// Auto-deactivate tokens that FCM reported as invalid.
		s.deactivateInvalidTokens(ctx, tokenStrings, resp)
	}

	// Log the notification.
	n := &repository.Notification{
		ID:            notifID,
		UserID:        req.GetUserId(),
		Title:         req.GetTitle(),
		Body:          req.GetBody(),
		Data:          req.GetData(),
		ImageURL:      req.GetImageUrl(),
		Status:        notifStatus,
		FailureReason: failureReason,
		SentAt:        &now,
	}
	saved, saveErr := s.repo.CreateNotification(ctx, n)
	if saveErr != nil {
		s.logger.Error("failed to log notification", zap.Error(saveErr))
	}

	s.logger.Info("notification sent",
		zap.String("notification_id", notifID),
		zap.String("user_id", req.GetUserId()),
		zap.Int32("devices_sent", devicesSent),
		zap.Int32("devices_failed", devicesFailed),
	)

	return &notifv1.SendNotificationResponse{
		Notification:  notificationToProto(saved),
		DevicesSent:   devicesSent,
		DevicesFailed: devicesFailed,
	}, nil
}

func (s *NotificationService) SendBulkNotification(ctx context.Context, req *notifv1.SendBulkNotificationRequest) (*notifv1.SendBulkNotificationResponse, error) {
	if len(req.GetUserIds()) == 0 || req.GetTitle() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_ids and title are required")
	}

	var (
		totalSent    int32
		totalFailed  int32
		notifications []*notifv1.Notification
	)

	for _, userID := range req.GetUserIds() {
		resp, err := s.SendNotification(ctx, &notifv1.SendNotificationRequest{
			UserId:   userID,
			Title:    req.GetTitle(),
			Body:     req.GetBody(),
			Data:     req.GetData(),
			ImageUrl: req.GetImageUrl(),
		})
		if err != nil {
			s.logger.Warn("bulk send failed for user", zap.String("user_id", userID), zap.Error(err))
			continue
		}
		totalSent += resp.GetDevicesSent()
		totalFailed += resp.GetDevicesFailed()
		if resp.GetNotification() != nil {
			notifications = append(notifications, resp.GetNotification())
		}
	}

	return &notifv1.SendBulkNotificationResponse{
		TotalUsers:         int32(len(req.GetUserIds())),
		TotalDevicesSent:   totalSent,
		TotalDevicesFailed: totalFailed,
		Notifications:      notifications,
	}, nil
}

func (s *NotificationService) GetNotification(ctx context.Context, req *notifv1.GetNotificationRequest) (*notifv1.GetNotificationResponse, error) {
	if req.GetNotificationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}

	n, err := s.repo.GetNotification(ctx, req.GetNotificationId())
	if err != nil {
		if errors.Is(err, repository.ErrNotificationNotFound) {
			return nil, status.Error(codes.NotFound, "notification not found")
		}
		s.logger.Error("failed to get notification", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get notification")
	}

	return &notifv1.GetNotificationResponse{Notification: notificationToProto(n)}, nil
}

func (s *NotificationService) ListNotifications(ctx context.Context, req *notifv1.ListNotificationsRequest) (*notifv1.ListNotificationsResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	notifications, nextPageToken, err := s.repo.ListNotifications(ctx, req.GetUserId(), pageSize, req.GetPageToken())
	if err != nil {
		s.logger.Error("failed to list notifications", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list notifications")
	}

	protoNotifs := make([]*notifv1.Notification, len(notifications))
	for i, n := range notifications {
		protoNotifs[i] = notificationToProto(n)
	}

	return &notifv1.ListNotificationsResponse{
		Notifications: protoNotifs,
		NextPageToken: nextPageToken,
	}, nil
}

// ---- helpers ----

// deactivateInvalidTokens marks tokens as inactive when FCM reports them as
// unregistered or invalid.
func (s *NotificationService) deactivateInvalidTokens(ctx context.Context, tokens []string, resp *fcm.BatchResponse) {
	var invalid []string
	for i, r := range resp.Results {
		if !r.Success && i < len(tokens) {
			invalid = append(invalid, tokens[i])
		}
	}
	if len(invalid) == 0 {
		return
	}
	if err := s.repo.DeactivateDeviceTokens(ctx, invalid); err != nil {
		s.logger.Warn("failed to deactivate invalid tokens", zap.Error(err))
	} else {
		s.logger.Info("deactivated invalid device tokens", zap.Int("count", len(invalid)))
	}
}

func deviceTokenToProto(dt *repository.DeviceToken) *notifv1.DeviceToken {
	if dt == nil {
		return nil
	}
	return &notifv1.DeviceToken{
		Id:         dt.ID,
		UserId:     dt.UserID,
		Token:      dt.Token,
		Platform:   dt.Platform,
		DeviceName: dt.DeviceName,
		Active:     dt.Active,
		CreatedAt:  timestamppb.New(dt.CreatedAt),
		UpdatedAt:  timestamppb.New(dt.UpdatedAt),
	}
}

func notificationToProto(n *repository.Notification) *notifv1.Notification {
	if n == nil {
		return nil
	}
	pn := &notifv1.Notification{
		Id:            n.ID,
		UserId:        n.UserID,
		Title:         n.Title,
		Body:          n.Body,
		Data:          n.Data,
		ImageUrl:      n.ImageURL,
		Status:        n.Status,
		FailureReason: n.FailureReason,
		CreatedAt:     timestamppb.New(n.CreatedAt),
	}
	if n.SentAt != nil {
		pn.SentAt = timestamppb.New(*n.SentAt)
	}
	return pn
}