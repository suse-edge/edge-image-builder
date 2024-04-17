package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"go.uber.org/zap"
)

// DownloadFile downloads a file from the specified URL and stores it to the given path.
//
// Optionally provide an additional cache writer in cases where the pending download
// must be stored to other locations alongside the given path.
func DownloadFile(ctx context.Context, url, path string, cache io.Writer) error {
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

	var writers []io.Writer
	writers = append(writers, file)

	if cache != nil {
		writers = append(writers, cache)
	}

	message := fmt.Sprintf("Downloading file: %s", filename)

	if resp.ContentLength == -1 {
		// Only audit the message since progress bars of unknown length
		// (i.e. spinners) are not properly rendered.
		log.Audit(message)
	} else {
		bar := progressbar.DefaultBytes(resp.ContentLength, message)
		writers = append(writers, bar)
	}

	if _, err = io.Copy(io.MultiWriter(writers...), resp.Body); err != nil {
		return fmt.Errorf("storing response: %w", err)
	}

	zap.S().Infof("Downloading file '%s' completed", filename)

	return nil
}
