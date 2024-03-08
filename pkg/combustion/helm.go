package combustion

import "github.com/suse-edge/edge-image-builder/pkg/image"

func ComponentHelmCharts(ctx *image.Context) []image.HelmChart {
	if ctx.ImageDefinition.Kubernetes.Version == "" {
		return nil
	}

	const (
		suseEdgeRepository    = "https://suse-edge.github.io/charts"
		installationNamespace = "kube-system"
	)

	var charts []image.HelmChart

	if ctx.ImageDefinition.Kubernetes.Network.APIVIP != "" {
		metalLBChart := image.HelmChart{
			Name:                  "metallb",
			Repo:                  suseEdgeRepository,
			TargetNamespace:       "metallb-system",
			CreateNamespace:       true,
			InstallationNamespace: installationNamespace,
			Version:               "0.14.3",
		}

		endpointCopierOperatorChart := image.HelmChart{
			Name:                  "endpoint-copier-operator",
			Repo:                  suseEdgeRepository,
			TargetNamespace:       "endpoint-copier-operator",
			CreateNamespace:       true,
			InstallationNamespace: installationNamespace,
			Version:               "0.2.0",
		}

		charts = append(charts, metalLBChart, endpointCopierOperatorChart)
	}

	return charts
}
