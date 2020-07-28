package uploader

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type Service interface {
	Upload(string, io.Reader) (string, error)
}

type impl struct {
	s3manager *s3manager.Uploader
}

func New(s3manager *s3manager.Uploader) Service {
	return &impl{s3manager}
}

func (s *impl) Upload(fileName string, r io.Reader) (string, error) {

	bucketName := "try-imager"
	aclPerm := "public-read"

	result, err := s.s3manager.Upload(&s3manager.UploadInput{
		Bucket: &bucketName,
		Key:    &fileName,
		Body:   r,
		ACL:    &aclPerm,
	})

	if err != nil {
		return "", fmt.Errorf("can't upload %s with error: %v", fileName, err)
	}

	return result.Location, nil
}