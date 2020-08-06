package uploader

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Service describes uploader interface.
type Service interface {
	Upload(context.Context, string, io.Reader) (string, error)
}

type impl struct {
	s3manager  *s3manager.Uploader
	bucketName *string
}

// New returns uploader implementation using s3 manager.
func New(s3manager *s3manager.Uploader, bucketName *string) Service {
	return &impl{s3manager, bucketName}
}

// Upload uploads image to s3 bucket and returns link for download.
func (s *impl) Upload(ctx context.Context, fileName string, r io.Reader) (string, error) {

	var (
		aclPerm = "public-read"
	)

	result, err := s.s3manager.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: s.bucketName,
		Key:    &fileName,
		Body:   r,
		ACL:    &aclPerm,
	})

	if err != nil {
		return "", fmt.Errorf("can't upload %s with error: %v", fileName, err)
	}

	return result.Location, nil
}
