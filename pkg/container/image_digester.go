package container

import (
	"fmt"
	"strings"

	"github.com/containers/image/v5/manifest"
)

type imageInspector interface {
	Inspect(image string) (*manifest.Schema2List, error)
}

type ImageDigester struct {
	ImageInspector imageInspector
}

func (d *ImageDigester) ImageDigest(img string, arch string) (string, error) {
	schemas, err := d.ImageInspector.Inspect(img)
	if err != nil {
		return "", fmt.Errorf("inspecting image: %w", err)
	}

	for _, m := range schemas.Manifests {
		if m.Platform.OS == "linux" && m.Platform.Architecture == arch {
			digest := m.Digest.String()
			// This is done to remove "sha256:" from the digest
			if parts := strings.Split(digest, "sha256:"); len(parts) == 2 {
				digest = parts[1]
			}
			return digest, nil
		}
	}

	return "", fmt.Errorf("image is not built for linux/%s", arch)
}
