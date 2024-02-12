package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSELinuxPolicy_K3s(t *testing.T) {
	policy := newSELinuxPolicy("v1.29.1+k3s2")

	expectedURL := "https://github.com/k3s-io/k3s-selinux/releases/download/v1.5.stable.1/k3s-selinux-1.5-1.slemicro.noarch.rpm"
	expectedRPM := "k3s-selinux-1.5-1.slemicro.noarch.rpm"

	assert.Equal(t, expectedURL, policy.downloadURL)
	assert.Equal(t, expectedRPM, policy.rpmName)
}

func TestNewSELinuxPolicy_RKE2(t *testing.T) {
	policy := newSELinuxPolicy("v1.29.0+rke2r1")

	expectedURL := "https://github.com/rancher/rke2-selinux/releases/download/v0.17.stable.1/rke2-selinux-0.17-1.slemicro.noarch.rpm"
	expectedRPM := "rke2-selinux-0.17-1.slemicro.noarch.rpm"

	assert.Equal(t, expectedURL, policy.downloadURL)
	assert.Equal(t, expectedRPM, policy.rpmName)
}
