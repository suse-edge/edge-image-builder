package container

import (
	"fmt"
	"github.com/suse-edge/edge-image-builder/pkg/log"
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
				log.Audit("WARNING: Container image with digest detected, please be that the digest is a manifest " +
					"digest (the digest of the container image version) and NOT an index digest (the digest of the " +
					"image platform). The embedded artifact registry will fail at boot time if an index/platform specific digest is provided.")
			}
			return digest, nil
		}
	}

	return "", fmt.Errorf("image is not built for linux/%s", arch)
}
