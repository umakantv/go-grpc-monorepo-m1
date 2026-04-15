package service

import (
	"context"
	"errors"

	"github.com/yourorg/monorepo/clients/auth-client"
	userv1 "github.com/yourorg/monorepo/gen/go/public/user"
	"github.com/yourorg/monorepo/services/user-api/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserService implements the UserService gRPC service
type UserService struct {
	userv1.UnimplementedUserServiceServer

	repo       *repository.Repository
	authClient *authclient.Client
	logger     *zap.Logger
}

// NewUserService creates a new UserService
func NewUserService(repo *repository.Repository, authClient *authclient.Client, logger *zap.Logger) *UserService {
	return &UserService{
		repo:       repo,
		authClient: authClient,
		logger:     logger.Named("user-service"),
	}
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	u, err := s.repo.GetByID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.logger.Error("failed to get user", zap.Error(err), zap.String("user_id", req.GetUserId()))
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return &userv1.GetUserResponse{
		User: u.ToProto(),
	}, nil
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	// Check if user already exists
	existing, err := s.repo.GetByEmail(ctx, req.GetEmail())
	if err == nil && existing != nil {
		return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
	}

	// In a real application, you would hash the password here
	passwordHash := req.GetPassword() // TODO: hash password

	u, err := s.repo.Create(ctx, req.GetEmail(), req.GetName(), passwordHash)
	if err != nil {
		s.logger.Error("failed to create user", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	s.logger.Info("user created", zap.String("user_id", u.ID), zap.String("email", u.Email))

	return &userv1.CreateUserResponse{
		User: &userv1.User{
			UserId:    u.ID,
			Email:     u.Email,
			Name:      u.Name,
			CreatedAt: timestamppb.New(u.CreatedAt),
			UpdatedAt: timestamppb.New(u.UpdatedAt),
		},
	}, nil
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	updates := make(map[string]interface{})

	if req.Email != nil {
		updates["email"] = req.GetEmail()
	}
	if req.Name != nil {
		updates["name"] = req.GetName()
	}

	if len(updates) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no updates provided")
	}

	u, err := s.repo.Update(ctx, req.GetUserId(), updates)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.logger.Error("failed to update user", zap.Error(err), zap.String("user_id", req.GetUserId()))
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	return &userv1.UpdateUserResponse{
		User: &userv1.User{
			UserId:    u.ID,
			Email:     u.Email,
			Name:      u.Name,
			CreatedAt: timestamppb.New(u.CreatedAt),
			UpdatedAt: timestamppb.New(u.UpdatedAt),
		},
	}, nil
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, req *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error) {
	err := s.repo.Delete(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.logger.Error("failed to delete user", zap.Error(err), zap.String("user_id", req.GetUserId()))
		return nil, status.Error(codes.Internal, "failed to delete user")
	}

	s.logger.Info("user deleted", zap.String("user_id", req.GetUserId()))

	return &userv1.DeleteUserResponse{}, nil
}

// ListUsers lists all users with pagination
func (s *UserService) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	users, nextPageToken, err := s.repo.List(ctx, pageSize, req.GetPageToken())
	if err != nil {
		s.logger.Error("failed to list users", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list users")
	}

	protoUsers := make([]*userv1.User, len(users))
	for i, u := range users {
		protoUsers[i] = &userv1.User{
			UserId:    u.ID,
			Email:     u.Email,
			Name:      u.Name,
			CreatedAt: timestamppb.New(u.CreatedAt),
			UpdatedAt: timestamppb.New(u.UpdatedAt),
		}
	}

	return &userv1.ListUsersResponse{
		Users:         protoUsers,
		NextPageToken: nextPageToken,
	}, nil
}
