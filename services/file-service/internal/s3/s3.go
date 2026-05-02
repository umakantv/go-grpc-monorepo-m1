package s3

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
)

// Client wraps the AWS S3 SDK for signed-URL operations.
type Client struct {
	svc             *awss3.S3
	bucket          string
	signedURLExpiry time.Duration
}

// Config holds the parameters needed to build an S3 client.
type Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey  string
	Endpoint        string
	ForcePathStyle  bool
	SignedURLExpiry int // seconds
}

// New creates an S3 client from the given configuration.
func New(cfg Config) (*Client, error) {
	awsCfg := &aws.Config{
		Region: aws.String(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		awsCfg.Credentials = credentials.NewStaticCredentials(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)
	}

	if cfg.Endpoint != "" {
		awsCfg.Endpoint = aws.String(cfg.Endpoint)
	}
	if cfg.ForcePathStyle {
		awsCfg.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession(awsCfg)
	if err != nil {
		return nil, fmt.Errorf("s3: failed to create session: %w", err)
	}

	expiry := time.Duration(cfg.SignedURLExpiry) * time.Second
	if expiry == 0 {
		expiry = 15 * time.Minute
	}

	return &Client{
		svc:             awss3.New(sess),
		bucket:          cfg.Bucket,
		signedURLExpiry: expiry,
	}, nil
}

// GeneratePutSignedURL returns a pre-signed PUT URL that the client can use
// to upload a file directly to S3.
func (c *Client) GeneratePutSignedURL(s3Key, mimeType string) (string, error) {
	req, _ := c.svc.PutObjectRequest(&awss3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(s3Key),
		ContentType: aws.String(mimeType),
	})

	url, err := req.Presign(c.signedURLExpiry)
	if err != nil {
		return "", fmt.Errorf("s3: failed to presign put url: %w", err)
	}
	return url, nil
}

// ObjectURL returns the public (or endpoint-based) URL for a given S3 key.
func (c *Client) ObjectURL(s3Key string) string {
	if c.svc.Endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", c.svc.Endpoint, c.bucket, s3Key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		c.bucket, *c.svc.Config.Region, s3Key)
}