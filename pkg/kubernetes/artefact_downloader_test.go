package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestGatherArtefacts(t *testing.T) {
	tests := []struct {
		name              string
		kubernetes        image.Kubernetes
		arch              image.Arch
		expectedArtefacts []string
		expectedError     string
	}{
		{
			name:          "CNI not specified",
			kubernetes:    image.Kubernetes{},
			arch:          image.ArchTypeX86,
			expectedError: "CNI not specified",
		},
		{
			name: "CNI not supported",
			kubernetes: image.Kubernetes{
				CNI: "flannel",
			},
			arch:          image.ArchTypeX86,
			expectedError: "unsupported CNI: flannel",
		},
		{
			name: "x86_64 artefacts without CNI",
			kubernetes: image.Kubernetes{
				CNI: image.CNITypeNone,
			},
			arch: image.ArchTypeX86,
			expectedArtefacts: []string{
				"rke2.linux-amd64.tar.gz",
				"rke2-images-core.linux-amd64.tar.zst",
				"sha256sum-amd64.txt",
			},
		},
		{
			name: "x86_64 artefacts with canal CNI",
			kubernetes: image.Kubernetes{
				CNI: image.CNITypeCanal,
			},
			arch: image.ArchTypeX86,
			expectedArtefacts: []string{
				"rke2.linux-amd64.tar.gz",
				"rke2-images-core.linux-amd64.tar.zst",
				"sha256sum-amd64.txt",
				"rke2-images-canal.linux-amd64.tar.zst",
			},
		},
		{
			name: "x86_64 artefacts with calico CNI",
			kubernetes: image.Kubernetes{
				CNI: image.CNITypeCalico,
			},
			arch: image.ArchTypeX86,
			expectedArtefacts: []string{
				"rke2.linux-amd64.tar.gz",
				"rke2-images-core.linux-amd64.tar.zst",
				"sha256sum-amd64.txt",
				"rke2-images-calico.linux-amd64.tar.zst",
			},
		},
		{
			name: "x86_64 artefacts with cilium CNI",
			kubernetes: image.Kubernetes{
				CNI: image.CNITypeCilium,
			},
			arch: image.ArchTypeX86,
			expectedArtefacts: []string{
				"rke2.linux-amd64.tar.gz",
				"rke2-images-core.linux-amd64.tar.zst",
				"sha256sum-amd64.txt",
				"rke2-images-cilium.linux-amd64.tar.zst",
			},
		},
		{
			name: "x86_64 artefacts with cilium CNI + multus + vSphere",
			kubernetes: image.Kubernetes{
				CNI:            image.CNITypeCilium,
				MultusEnabled:  true,
				VSphereEnabled: true,
			},
			arch: image.ArchTypeX86,
			expectedArtefacts: []string{
				"rke2.linux-amd64.tar.gz",
				"rke2-images-core.linux-amd64.tar.zst",
				"sha256sum-amd64.txt",
				"rke2-images-cilium.linux-amd64.tar.zst",
				"rke2-images-multus.linux-amd64.tar.zst",
				"rke2-images-vsphere.linux-amd64.tar.zst",
			},
		},
		{
			name: "aarch64 artefacts for CNI none",
			kubernetes: image.Kubernetes{
				CNI: image.CNITypeNone,
			},
			arch: image.ArchTypeARM,
			expectedArtefacts: []string{
				"rke2.linux-arm64.tar.gz",
				"rke2-images-core.linux-arm64.tar.zst",
				"sha256sum-arm64.txt",
			},
		},
		{
			name: "aarch64 artefacts with canal CNI",
			kubernetes: image.Kubernetes{
				CNI: image.CNITypeCanal,
			},
			arch: image.ArchTypeARM,
			expectedArtefacts: []string{
				"rke2.linux-arm64.tar.gz",
				"rke2-images-core.linux-arm64.tar.zst",
				"sha256sum-arm64.txt",
				"rke2-images-canal.linux-arm64.tar.zst",
			},
		},
		{
			name: "aarch64 artefacts with calico CNI",
			kubernetes: image.Kubernetes{
				CNI: image.CNITypeCalico,
			},
			arch:          image.ArchTypeARM,
			expectedError: "calico is not supported on aarch64 platforms",
		},
		{
			name: "aarch64 artefacts with cilium CNI",
			kubernetes: image.Kubernetes{
				CNI: image.CNITypeCilium,
			},
			arch:          image.ArchTypeARM,
			expectedError: "cilium is not supported on aarch64 platforms",
		},
		{
			name: "aarch64 artefacts with canal CNI + multus",
			kubernetes: image.Kubernetes{
				CNI:           image.CNITypeCanal,
				MultusEnabled: true,
			},
			arch:          image.ArchTypeARM,
			expectedError: "multus is not supported on aarch64 platforms",
		},
		{
			name: "aarch64 artefacts with canal CNI + vSphere",
			kubernetes: image.Kubernetes{
				CNI:            image.CNITypeCanal,
				VSphereEnabled: true,
			},
			arch:          image.ArchTypeARM,
			expectedError: "vSphere is not supported on aarch64 platforms",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artefacts, err := gatherArtefacts(test.kubernetes, test.arch)

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
