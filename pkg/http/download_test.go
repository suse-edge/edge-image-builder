package http

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadFile(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		url         string
		path        string
		expectedErr string
	}{
		{
			name:        "Nil context",
			expectedErr: "creating request: net/http: nil Context",
		},
		{
			name:        "Invalid URL",
			ctx:         context.Background(),
			url:         "invalid-url",
			expectedErr: "executing request: Get \"invalid-url\": unsupported protocol scheme \"\"",
		},
		{
			name:        "Unexpected status",
			ctx:         context.Background(),
			url:         "https://github.com/suse-edge/eib",
			expectedErr: "unexpected status code: 404",
		},
		{
			name:        "Error creating file",
			ctx:         context.Background(),
			url:         "https://github.com/suse-edge/edge-image-builder",
			path:        "downloads/abc",
			expectedErr: "creating file: open downloads/abc: no such file or directory",
		},
		{
			name: "Successful download",
			ctx:  context.Background(),
			url:  "https://raw.githubusercontent.com/suse-edge/edge-image-builder/main/README.md",
			path: "README.md",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := DownloadFile(test.ctx, test.url, test.path)
			if test.expectedErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)

				_, err := os.Stat(test.path)
				assert.NoError(t, err)
				assert.NoError(t, os.Remove(test.path))
			}
		})
	}
}
