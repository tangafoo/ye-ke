// Package blob stores files we serve ourselves in Cloudflare R2 (S3-compatible)
// and hands back their public URL. Used for street photos so their links never
// expire (Mapillary's own URLs are signed and do).
package blob

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Store struct {
	s3        *s3.Client
	bucket    string
	publicURL string // e.g. https://img.yeke.app or https://pub-xxxx.r2.dev
}

type Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicBaseURL   string
}

// New returns nil when any setting is missing, so callers treat R2 as off.
func New(c Config) *Store {
	if c.AccountID == "" || c.AccessKeyID == "" || c.SecretAccessKey == "" || c.Bucket == "" || c.PublicBaseURL == "" {
		return nil
	}
	return newWithEndpoint(c, fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.AccountID))
}

func newWithEndpoint(c Config, endpoint string) *Store {
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(endpoint),
		Region:       "auto", // R2 ignores regions; the SDK requires one
		Credentials:  credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, ""),
		UsePathStyle: true,
	})
	return &Store{s3: client, bucket: c.Bucket, publicURL: strings.TrimRight(c.PublicBaseURL, "/")}
}

// Put uploads body under key and returns its public URL. Objects are
// immutable (a key is never reused for different bytes), so they're cached
// for a year by browsers and Cloudflare.
func (s *Store) Put(ctx context.Context, key, contentType string, body []byte) (string, error) {
	_, err := s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(body),
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return "", fmt.Errorf("[blob] put %s: %w", key, err)
	}
	return s.URL(key), nil
}

func (s *Store) URL(key string) string { return s.publicURL + "/" + key }
