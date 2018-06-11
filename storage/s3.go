package storage

import (
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
)

type S3Client interface {
	PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

type S3 struct {
	client S3Client
	bucket string
}

func (s *S3) UploadFile(path string, r io.ReadSeeker) error {
	po := &s3.PutObjectInput{Bucket: &s.bucket, Body: r, Key: &path}
	_, err := s.client.PutObject(po)
	return errors.Wrapf(err, "failed to upload file %s", path)
}

func newS3(cfg *Config) *S3 {
	st := &S3{bucket: cfg.Bucket}
	awsCfg := &aws.Config{
		Credentials: credentials.NewStaticCredentials(
			cfg.Key,
			cfg.Secret,
			"",
		),
		Region: &cfg.Region,
	}
	st.client = s3.New(session.New(), awsCfg)
	return st
}
