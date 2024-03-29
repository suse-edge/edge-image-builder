package combustion

import (
	"github.com/suse-edge/edge-image-builder/pkg/env"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func ComponentHelmCharts(ctx *image.Context) ([]image.HelmChart, []image.HelmRepository) {
	if ctx.ImageDefinition.Kubernetes.Version == "" {
		return nil, nil
	}

	const (
		suseEdgeRepositoryName = "suse-edge"
		installationNamespace  = "kube-system"
	)

	var charts []image.HelmChart
	var repos []image.HelmRepository

	if ctx.ImageDefinition.Kubernetes.Network.APIVIP != "" {
		metalLBChart := image.HelmChart{
			Name:                  "metallb",
			RepositoryName:        suseEdgeRepositoryName,
			TargetNamespace:       "metallb-system",
			CreateNamespace:       true,
			InstallationNamespace: installationNamespace,
			Version:               "0.14.3",
		}

		endpointCopierOperatorChart := image.HelmChart{
			Name:                  "endpoint-copier-operator",
			RepositoryName:        suseEdgeRepositoryName,
			TargetNamespace:       "endpoint-copier-operator",
			CreateNamespace:       true,
			InstallationNamespace: installationNamespace,
			Version:               "0.2.0",
		}

		charts = append(charts, metalLBChart, endpointCopierOperatorChart)

		suseEdgeRepo := image.HelmRepository{
			Name: suseEdgeRepositoryName,
			URL:  env.EdgeHelmRepository,
		}

		repos = append(repos, suseEdgeRepo)
	}

	return charts, repos
}
