package downloader

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Service describes donwloader interface.
type Service interface {
	Download(context.Context, string) ([]byte, error)
}

type impl struct{}

// New returns downloader implementation.
func New() Service {
	return &impl{}
}

// Download downloads file and returns response body from it.
func (s *impl) Download(ctx context.Context, url string) ([]byte, error) {
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

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error downloading %s, status code is: %d", url, res.StatusCode)
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body for: %s failed with error: %v", url, err)
	}

	return b, nil
}
