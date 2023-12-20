package combustion

import (
	"fmt"

	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
)

const (
	k8sComponentName = "kubernetes"
)

func configureKubernetes(ctx *image.Context) ([]string, error) {
	switch ctx.ImageDefinition.Kubernetes {
	case "":
		log.AuditComponentSkipped(k8sComponentName)
		return nil, nil
	case image.KubernetesTypeRKE2:
		panic("implement me")
	case image.KubernetesTypeK3s:
		panic("implement me")
	default:
		log.AuditComponentFailed(k8sComponentName)
		return nil, fmt.Errorf("unexpected k8s distro: %s", ctx.ImageDefinition.Kubernetes)
	}
}
