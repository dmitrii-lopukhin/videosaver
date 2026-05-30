package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrTooLarge is returned when the response body exceeds maxBytes.
var ErrTooLarge = errors.New("download: response exceeds size limit")

// Fetch makes a GET request to directURL and returns a ReadCloser limited to
// maxBytes. The caller is responsible for closing the returned reader.
func Fetch(ctx context.Context, directURL string, maxBytes int64) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, directURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download: HTTP %d from %s", resp.StatusCode, directURL)
	}

	if resp.ContentLength > 0 && resp.ContentLength > maxBytes {
		resp.Body.Close()
		return nil, ErrTooLarge
	}

	return &limitedReadCloser{ReadCloser: resp.Body, limit: maxBytes}, nil
}

type limitedReadCloser struct {
	io.ReadCloser
	limit int64
	read  int64
}

func (l *limitedReadCloser) Read(p []byte) (int, error) {
	if l.read >= l.limit {
		return 0, ErrTooLarge
	}
	if rem := l.limit - l.read; int64(len(p)) > rem {
		p = p[:rem]
	}
	n, err := l.ReadCloser.Read(p)
	l.read += int64(n)
	if l.read >= l.limit && err == nil {
		err = ErrTooLarge
	}
	return n, err
}
