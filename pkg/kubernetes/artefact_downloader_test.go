package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestRKE2InstallerArtefacts(t *testing.T) {
	x86Artefacts := []string{"rke2.linux-amd64.tar.gz", "sha256sum-amd64.txt"}
	assert.Equal(t, x86Artefacts, rke2InstallerArtefacts(image.ArchTypeX86))

	armArtefacts := []string{"rke2.linux-arm64.tar.gz", "sha256sum-arm64.txt"}
	assert.Equal(t, armArtefacts, rke2InstallerArtefacts(image.ArchTypeARM))
}

func TestRKE2ImageArtefacts(t *testing.T) {
	tests := []struct {
		name              string
		cni               string
		multusEnabled     bool
		arch              image.Arch
		expectedArtefacts []string
		expectedError     string
	}{
		{
			name:          "CNI not specified",
			arch:          image.ArchTypeX86,
			expectedError: "CNI not specified",
		},
		{
			name:          "CNI not supported",
			cni:           "flannel",
			arch:          image.ArchTypeX86,
			expectedError: "unsupported CNI: flannel",
		},
		{
			name: "x86_64 artefacts without CNI",
			cni:  image.CNITypeNone,
			arch: image.ArchTypeX86,
			expectedArtefacts: []string{
				"rke2-images-core.linux-amd64.tar.zst",
			},
		},
		{
			name: "x86_64 artefacts with canal CNI",
			cni:  image.CNITypeCanal,
			arch: image.ArchTypeX86,
			expectedArtefacts: []string{
				"rke2-images-core.linux-amd64.tar.zst",
				"rke2-images-canal.linux-amd64.tar.zst",
			},
		},
		{
			name: "x86_64 artefacts with calico CNI",
			cni:  image.CNITypeCalico,
			arch: image.ArchTypeX86,
			expectedArtefacts: []string{
				"rke2-images-core.linux-amd64.tar.zst",
				"rke2-images-calico.linux-amd64.tar.zst",
			},
		},
		{
			name: "x86_64 artefacts with cilium CNI",
			cni:  image.CNITypeCilium,
			arch: image.ArchTypeX86,
			expectedArtefacts: []string{
				"rke2-images-core.linux-amd64.tar.zst",
				"rke2-images-cilium.linux-amd64.tar.zst",
			},
		},
		{
			name:          "x86_64 artefacts with cilium CNI + multus",
			cni:           image.CNITypeCilium,
			multusEnabled: true,
			arch:          image.ArchTypeX86,
			expectedArtefacts: []string{
				"rke2-images-core.linux-amd64.tar.zst",
				"rke2-images-cilium.linux-amd64.tar.zst",
				"rke2-images-multus.linux-amd64.tar.zst",
			},
		},
		{
			name: "aarch64 artefacts for CNI none",
			cni:  image.CNITypeNone,
			arch: image.ArchTypeARM,
			expectedArtefacts: []string{
				"rke2-images-core.linux-arm64.tar.zst",
			},
		},
		{
			name: "aarch64 artefacts with canal CNI",
			cni:  image.CNITypeCanal,
			arch: image.ArchTypeARM,
			expectedArtefacts: []string{
				"rke2-images-core.linux-arm64.tar.zst",
				"rke2-images-canal.linux-arm64.tar.zst",
			},
		},
		{
			name:          "aarch64 artefacts with calico CNI",
			cni:           image.CNITypeCalico,
			arch:          image.ArchTypeARM,
			expectedError: "calico is not supported on aarch64 platforms",
		},
		{
			name:          "aarch64 artefacts with cilium CNI",
			cni:           image.CNITypeCilium,
			arch:          image.ArchTypeARM,
			expectedError: "cilium is not supported on aarch64 platforms",
		},
		{
			name:          "aarch64 artefacts with canal CNI + multus",
			cni:           image.CNITypeCanal,
			multusEnabled: true,
			arch:          image.ArchTypeARM,
			expectedError: "multus is not supported on aarch64 platforms",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artefacts, err := rke2ImageArtefacts(test.cni, test.multusEnabled, test.arch)

			if test.expectedError != "" {
				require.EqualError(t, err, test.expectedError)
				assert.Nil(t, artefacts)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectedArtefacts, artefacts)
			}
		})
	}
}

func TestK3sInstallerArtefacts(t *testing.T) {
	x86Artefacts := []string{"k3s"}
	assert.Equal(t, x86Artefacts, k3sInstallerArtefacts(image.ArchTypeX86))

	armArtefacts := []string{"k3s-arm64"}
	assert.Equal(t, armArtefacts, k3sInstallerArtefacts(image.ArchTypeARM))
}

func TestK3sImageArtefacts(t *testing.T) {
	x86Artefacts := []string{"k3s-airgap-images-amd64.tar.zst"}
	assert.Equal(t, x86Artefacts, k3sImageArtefacts(image.ArchTypeX86))

	armArtefacts := []string{"k3s-airgap-images-arm64.tar.zst"}
	assert.Equal(t, armArtefacts, k3sImageArtefacts(image.ArchTypeARM))
}
