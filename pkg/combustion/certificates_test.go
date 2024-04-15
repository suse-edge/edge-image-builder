package combustion

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func setupCertificatesConfigDir(t *testing.T) (ctx *image.Context, teardown func()) {
	ctx, teardown = setupContext(t)

	testCertsDir := filepath.Join(ctx.ImageConfigDir, certsConfigDir)
	err := os.Mkdir(testCertsDir, 0o755)
	require.NoError(t, err)

	testFilenames := []string{"foo", "bar.pem", "baz.pem", "wombat.crt"}
	for _, filename := range testFilenames {
		path := filepath.Join(testCertsDir, filename)
		err = os.WriteFile(path, []byte(""), 0o600)
		require.NoError(t, err)
	}

	return
}

func TestCopyCertificatesEmptyDirectory(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	testCertsDir := filepath.Join(ctx.ImageConfigDir, certsConfigDir)
	err := os.Mkdir(testCertsDir, 0o755)
	require.NoError(t, err)
	defer os.RemoveAll(testCertsDir)

	// Test
	err = copyCertificates(ctx)

	// Verify
	require.Error(t, err)
}

func TestCopyCertificates(t *testing.T) {
	// Setup
	ctx, teardown := setupCertificatesConfigDir(t)
	defer teardown()

	// Test
	err := copyCertificates(ctx)

	// Verify
	require.NoError(t, err)

	expectedCertsDir := filepath.Join(ctx.CombustionDir, certsConfigDir)
	expectedFilenames := []string{"bar.pem", "baz.pem", "wombat.crt"}
	entries, err := os.ReadDir(expectedCertsDir)
	require.NoError(t, err)
	assert.Len(t, entries, len(expectedFilenames))
	for _, entry := range entries {
		assert.Contains(t, expectedFilenames, entry.Name())
	}
}

func TestWriteCertificatesScript(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	err := writeCertificatesScript(ctx)

	// Verify
	require.NoError(t, err)

	scriptFilename := filepath.Join(ctx.CombustionDir, certsScriptName)
	foundBytes, err := os.ReadFile(scriptFilename)
	require.NoError(t, err)
	found := string(foundBytes)
	assert.Contains(t, found, fmt.Sprintf("cp ./%s/* /etc/pki/trust/anchors/.", certsConfigDir))
	assert.Contains(t, found, "update-ca-certificates")
}
