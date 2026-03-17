package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// GeneratePresignedPutURL creates a presigned S3 PUT URL for log upload
func GeneratePresignedPutURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	// Load AWS config (auto-detects credentials from env, file, or IRSA)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(client)

	req, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return req.URL, nil
}

// GenerateLogKey creates an S3 key in format: 2026-03-17T15-30-00Z/node-name/logs.tar.gz
func GenerateLogKey(nodeName string) string {
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	return fmt.Sprintf("%s/%s/logs.tar.gz", timestamp, nodeName)
}
