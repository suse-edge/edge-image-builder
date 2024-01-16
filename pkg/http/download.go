package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
	"go.uber.org/zap"
)

// DownloadFile downloads a file from the specified URL and stores it to the given path.
func DownloadFile(ctx context.Context, url, path string) error {
	filename := filepath.Base(path)

	zap.S().Infof("Downloading file '%s' from '%s' to '%s'...", filename, url, filepath.Dir(path))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	bar := progressbar.DefaultBytes(resp.ContentLength, fmt.Sprintf("Downloading file: %s", filename))

	if _, err = io.Copy(io.MultiWriter(file, bar), resp.Body); err != nil {
		return fmt.Errorf("storing response: %w", err)
	}

	zap.S().Infof("Downloading file '%s' completed", filename)

	return nil
}
