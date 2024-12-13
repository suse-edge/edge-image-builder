package combustion

import (
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func ComponentHelmCharts(ctx *image.Context) ([]image.HelmChart, []image.HelmRepository) {
	if ctx.ImageDefinition.Kubernetes.Version == "" {
		return nil, nil
	}

	const (
		metallbRepositoryName = "suse-edge-metallb"
		metallbNamespace      = "metallb-system"

		endpointCopierOperatorRepositoryName = "suse-edge-endpoint-copier-operator"
		endpointCopierOperatorNamespace      = "endpoint-copier-operator"

		installationNamespace = "kube-system"
	)

	var charts []image.HelmChart
	var repos []image.HelmRepository

	if ctx.ImageDefinition.Kubernetes.Network.APIVIP4 != "" || ctx.ImageDefinition.Kubernetes.Network.APIVIP6 != "" {
		metalLBChart := image.HelmChart{
			Name:                  ctx.ArtifactSources.MetalLB.Chart,
			RepositoryName:        metallbRepositoryName,
			TargetNamespace:       metallbNamespace,
			CreateNamespace:       true,
			InstallationNamespace: installationNamespace,
			Version:               ctx.ArtifactSources.MetalLB.Version,
		}

		endpointCopierOperatorChart := image.HelmChart{
			Name:                  ctx.ArtifactSources.EndpointCopierOperator.Chart,
			RepositoryName:        endpointCopierOperatorRepositoryName,
			TargetNamespace:       endpointCopierOperatorNamespace,
			CreateNamespace:       true,
			InstallationNamespace: installationNamespace,
			Version:               ctx.ArtifactSources.EndpointCopierOperator.Version,
		}

		charts = append(charts, metalLBChart, endpointCopierOperatorChart)

		metallbRepo := image.HelmRepository{
			Name: metallbRepositoryName,
			URL:  ctx.ArtifactSources.MetalLB.Repository,
		}

		endpointCopierOperatorRepo := image.HelmRepository{
			Name: endpointCopierOperatorRepositoryName,
			URL:  ctx.ArtifactSources.EndpointCopierOperator.Repository,
		}

		repos = append(repos, metallbRepo, endpointCopierOperatorRepo)
	}

	return charts, repos
}
