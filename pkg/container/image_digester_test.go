package container

import (
	"fmt"
	"testing"

	"github.com/containers/image/v5/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockImageInspector struct {
	inspect func(image string) (*manifest.Schema2List, error)
}

func (m mockImageInspector) Inspect(img string) (*manifest.Schema2List, error) {
	if m.inspect != nil {
		return m.inspect(img)
	}

	panic("not implemented")
}

func TestImageDigestValid(t *testing.T) {
	helloWorldManifest := &manifest.Schema2List{
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.list.v2+json",
		Manifests: []manifest.Schema2ManifestDescriptor{
			{
				Schema2Descriptor: manifest.Schema2Descriptor{
					MediaType: "application/vnd.docker.distribution.manifest.v2+json",
					Size:      1156,
					Digest:    "sha256:3dfc05677ed97fdf620a3af556d6fe44ec3747262cbf1b4c0c20eed284fd7290",
					URLs:      []string{},
				},
				Platform: manifest.Schema2PlatformSpec{
					Architecture: "amd64",
					OS:           "linux",
					OSVersion:    "",
					OSFeatures:   []string{},
					Variant:      "",
					Features:     []string{},
				},
			},
			{
				Schema2Descriptor: manifest.Schema2Descriptor{
					MediaType: "application/vnd.docker.distribution.manifest.v2+json",
					Size:      1156,
					Digest:    "sha256:7c831ce05c671702726fc2951fe85048b0b9559f4105b80363424aa935bff2d1",
					URLs:      []string{},
				},
				Platform: manifest.Schema2PlatformSpec{
					Architecture: "arm64",
					OS:           "linux",
					OSVersion:    "",
					OSFeatures:   []string{},
					Variant:      "",
					Features:     []string{},
				},
			},
		},
	}

	expectedDigest := "3dfc05677ed97fdf620a3af556d6fe44ec3747262cbf1b4c0c20eed284fd7290"

	d := ImageDigester{
		ImageInspector: mockImageInspector{
			inspect: func(image string) (*manifest.Schema2List, error) {
				return helloWorldManifest, nil
			},
		},
	}

	digest, err := d.ImageDigest("hello-world:latest", "amd64")
	require.NoError(t, err)
	assert.Equal(t, expectedDigest, digest)
}

func TestImageDigestNoSchemaFound(t *testing.T) {
	helloWorldManifest := &manifest.Schema2List{}

	d := ImageDigester{
		ImageInspector: mockImageInspector{
			inspect: func(image string) (*manifest.Schema2List, error) {
				return helloWorldManifest, nil
			},
		},
	}

	digest, err := d.ImageDigest("hello-world:latest", "amd64")
	require.NoError(t, err)
	assert.Empty(t, digest)
}

func TestImageDigestError(t *testing.T) {
	d := ImageDigester{
		ImageInspector: mockImageInspector{
			inspect: func(image string) (*manifest.Schema2List, error) {
				return nil, fmt.Errorf("image not found")
			},
		},
	}

	digest, err := d.ImageDigest("hello-world:latest", "amd64")
	require.ErrorContains(t, err, "image not found")
	assert.Empty(t, digest)
}
