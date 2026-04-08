package helper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

const maxImageSize = 5 * 1024 * 1024

var allowedImageMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

func UploadImageToS3(ctx context.Context, file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
	if header == nil {
		return "", errors.New("file header is required")
	}
	if header.Size > maxImageSize {
		return "", errors.New("file size must be less than or equal to 5MB")
	}

	data, err := io.ReadAll(io.LimitReader(file, maxImageSize+1))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", errors.New("file is empty")
	}
	if len(data) > maxImageSize {
		return "", errors.New("file size must be less than or equal to 5MB")
	}

	detectedMimeType := http.DetectContentType(data)
	ext, ok := allowedImageMimeTypes[detectedMimeType]
	if !ok {
		return "", errors.New("only jpeg, png and webp images are allowed")
	}

	s3Bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
	s3Region := strings.TrimSpace(os.Getenv("AWS_REGION"))
	if s3Bucket == "" || s3Region == "" {
		return "", errors.New("aws s3 configuration is missing")
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(s3Region))
	if err != nil {
		return "", err
	}

	client := s3.NewFromConfig(cfg)

	baseName := sanitizeFileName(header.Filename)
	fileName := fmt.Sprintf("%s-%s%s", uuid.NewString(), baseName, ext)
	objectKey := strings.Trim(strings.TrimSpace(folder), "/") + "/" + fileName

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s3Bucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(detectedMimeType),
	})
	if err != nil {
		return "", err
	}

	publicURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s3Bucket, s3Region, objectKey)
	return publicURL, nil
}

func sanitizeFileName(name string) string {
	base := strings.TrimSpace(filepath.Base(name))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.ToLower(base)
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.ReplaceAll(base, "/", "-")
	base = strings.ReplaceAll(base, "\\", "-")
	if base == "" {
		return "image"
	}
	return base
}
