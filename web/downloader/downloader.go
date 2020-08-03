package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type Service interface {
	Download(context.Context, string) (io.Reader, error)
}

type impl struct{}

func New() Service {
	return &impl{}
}

func (s *impl) Download(ctx context.Context, url string) (io.Reader, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.WithContext(ctx)
	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error downloading %s, status code is: %d", url, res.StatusCode)
	}

	return res.Body, nil
}
