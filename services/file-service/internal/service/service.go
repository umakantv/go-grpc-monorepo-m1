package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	filev1 "github.com/yourorg/monorepo/gen/go/public/file"
	"github.com/yourorg/monorepo/services/file-service/internal/repository"
	"github.com/yourorg/monorepo/services/file-service/internal/s3"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FileService implements the FileService gRPC service.
type FileService struct {
	filev1.UnimplementedFileServiceServer

	repo   *repository.Repository
	s3     *s3.Client
	logger *zap.Logger
}

// NewFileService creates a new FileService.
func NewFileService(repo *repository.Repository, s3Client *s3.Client, logger *zap.Logger) *FileService {
	return &FileService{
		repo:   repo,
		s3:     s3Client,
		logger: logger.Named("file-service"),
	}
}

// CreateUploadSignedURL generates a pre-signed S3 PUT URL and creates a pending file record.
func (s *FileService) CreateUploadSignedURL(ctx context.Context, req *filev1.CreateUploadSignedURLRequest) (*filev1.CreateUploadSignedURLResponse, error) {
	if req.GetFilename() == "" {
		return nil, status.Error(codes.InvalidArgument, "filename is required")
	}
	if req.GetMimetype() == "" {
		return nil, status.Error(codes.InvalidArgument, "mimetype is required")
	}
	if req.GetOwnerEntity() == "" || req.GetOwnerEntityId() == "" {
		return nil, status.Error(codes.InvalidArgument, "owner_entity and owner_entity_id are required")
	}

	fileID := uuid.New().String()

	// Build the S3 key: location/uuid/filename
	location := req.GetLocation()
	if location == "" {
		location = "uploads"
	}
	s3Key := fmt.Sprintf("%s/%s/%s", location, fileID, req.GetFilename())

	// Generate the pre-signed PUT URL.
	signedURL, err := s.s3.GeneratePutSignedURL(s3Key, req.GetMimetype())
	if err != nil {
		s.logger.Error("failed to generate signed url", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate upload url")
	}

	// Build the final object URL.
	objectURL := s.s3.ObjectURL(s3Key)

	// Create the file record in pending status.
	f := &repository.File{
		ID:            fileID,
		Filename:      req.GetFilename(),
		Mimetype:      req.GetMimetype(),
		Location:      location,
		SizeKB:        req.GetSizeKb(),
		OwnerEntity:   req.GetOwnerEntity(),
		OwnerEntityID: req.GetOwnerEntityId(),
		Status:        repository.StatusPending,
		S3Key:         s3Key,
		URL:           objectURL,
	}

	created, err := s.repo.Create(ctx, f)
	if err != nil {
		s.logger.Error("failed to create file record", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create file record")
	}

	s.logger.Info("upload signed url generated",
		zap.String("file_id", created.ID),
		zap.String("s3_key", s3Key),
	)

	return &filev1.CreateUploadSignedURLResponse{
		File:      fileToProto(created),
		SignedUrl: signedURL,
	}, nil
}

// ConfirmUpload marks a file as successfully uploaded.
func (s *FileService) ConfirmUpload(ctx context.Context, req *filev1.ConfirmUploadRequest) (*filev1.ConfirmUploadResponse, error) {
	if req.GetFileId() == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}

	f, err := s.repo.ConfirmUpload(ctx, req.GetFileId())
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		s.logger.Error("failed to confirm upload", zap.Error(err), zap.String("file_id", req.GetFileId()))
		return nil, status.Error(codes.Internal, "failed to confirm upload")
	}

	s.logger.Info("upload confirmed", zap.String("file_id", f.ID))

	return &filev1.ConfirmUploadResponse{
		File: fileToProto(f),
	}, nil
}

// GetFile retrieves file metadata by ID.
func (s *FileService) GetFile(ctx context.Context, req *filev1.GetFileRequest) (*filev1.GetFileResponse, error) {
	f, err := s.repo.GetByID(ctx, req.GetFileId())
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		s.logger.Error("failed to get file", zap.Error(err), zap.String("file_id", req.GetFileId()))
		return nil, status.Error(codes.Internal, "failed to get file")
	}

	return &filev1.GetFileResponse{
		File: fileToProto(f),
	}, nil
}

// DeleteFile soft-deletes a file.
func (s *FileService) DeleteFile(ctx context.Context, req *filev1.DeleteFileRequest) (*filev1.DeleteFileResponse, error) {
	err := s.repo.SoftDelete(ctx, req.GetFileId())
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, status.Error(codes.NotFound, "file not found")
		}
		s.logger.Error("failed to delete file", zap.Error(err), zap.String("file_id", req.GetFileId()))
		return nil, status.Error(codes.Internal, "failed to delete file")
	}

	s.logger.Info("file deleted", zap.String("file_id", req.GetFileId()))

	return &filev1.DeleteFileResponse{}, nil
}

// ListFiles lists files with optional owner filtering.
func (s *FileService) ListFiles(ctx context.Context, req *filev1.ListFilesRequest) (*filev1.ListFilesResponse, error) {
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	files, nextPageToken, err := s.repo.List(ctx, req.GetOwnerEntity(), req.GetOwnerEntityId(), pageSize, req.GetPageToken())
	if err != nil {
		s.logger.Error("failed to list files", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list files")
	}

	protoFiles := make([]*filev1.File, len(files))
	for i, f := range files {
		protoFiles[i] = fileToProto(f)
	}

	return &filev1.ListFilesResponse{
		Files:         protoFiles,
		NextPageToken: nextPageToken,
	}, nil
}

func fileToProto(f *repository.File) *filev1.File {
	pf := &filev1.File{
		Id:            f.ID,
		Filename:      f.Filename,
		Mimetype:      f.Mimetype,
		Location:      f.Location,
		SizeKb:        f.SizeKB,
		OwnerEntity:   f.OwnerEntity,
		OwnerEntityId: f.OwnerEntityID,
		Status:        f.Status,
		S3Key:         f.S3Key,
		Url:           f.URL,
		CreatedAt:     timestamppb.New(f.CreatedAt),
		UpdatedAt:     timestamppb.New(f.UpdatedAt),
	}
	return pf
}