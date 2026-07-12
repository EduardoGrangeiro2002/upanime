package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var knownContentTypes = map[string]string{
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".mkv":  "video/x-matroska",
	".avi":  "video/x-msvideo",
	".mov":  "video/quicktime",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
}

func contentTypeForKey(key string) string {
	ext := strings.ToLower(filepath.Ext(key))
	if ct, ok := knownContentTypes[ext]; ok {
		return ct
	}
	if ct := mime.TypeByExtension(ext); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

type R2Storage struct {
	client   *s3.Client
	presign  *s3.PresignClient
	uploader *manager.Uploader
	bucket   string
}

func NewR2Storage(accountID, accessKeyID, accessSecret, bucket string) *R2Storage {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	client := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(accessKeyID, accessSecret, ""),
	})

	return &R2Storage{
		client:   client,
		presign:  s3.NewPresignClient(client),
		uploader: manager.NewUploader(client),
		bucket:   bucket,
	}
}

func (s *R2Storage) Save(ctx context.Context, key string, reader io.Reader) error {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentTypeForKey(key)),
	})
	if err != nil {
		return fmt.Errorf("r2 upload: %w", err)
	}

	return nil
}

func (s *R2Storage) Download(ctx context.Context, key string, destPath string) error {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("r2 get object: %w", err)
	}
	defer resp.Body.Close()

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create dest file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("r2 download copy: %w", err)
	}

	return nil
}

func (s *R2Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("r2 delete: %w", err)
	}
	return nil
}

func (s *R2Storage) URL(ctx context.Context, key string) (string, error) {
	input := &s3.GetObjectInput{
		Bucket:              aws.String(s.bucket),
		Key:                 aws.String(key),
		ResponseContentType: aws.String(contentTypeForKey(key)),
	}

	presigned, err := s.presign.PresignGetObject(ctx, input, s3.WithPresignExpires(24*time.Hour))
	if err != nil {
		return "", fmt.Errorf("r2 presign: %w", err)
	}
	return presigned.URL, nil
}

func (s *R2Storage) PresignPutURL(ctx context.Context, key string) (string, error) {
	presigned, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentTypeForKey(key)),
	}, s3.WithPresignExpires(6*time.Hour))
	if err != nil {
		return "", fmt.Errorf("r2 presign put: %w", err)
	}
	return presigned.URL, nil
}

func (s *R2Storage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (s *R2Storage) ServeFile(ctx context.Context, w http.ResponseWriter, r *http.Request, key string) error {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		input.Range = aws.String(rangeHeader)
	}

	resp, err := s.client.GetObject(ctx, input)
	if err != nil {
		return fmt.Errorf("r2 get object: %w", err)
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", contentTypeForKey(key))
	w.Header().Set("Accept-Ranges", "bytes")

	if resp.ContentLength != nil {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", *resp.ContentLength))
	}

	if resp.ContentRange != nil {
		w.Header().Set("Content-Range", *resp.ContentRange)
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	io.Copy(w, resp.Body)
	return nil
}

func (s *R2Storage) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("r2 list: %w", err)
		}
		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}
