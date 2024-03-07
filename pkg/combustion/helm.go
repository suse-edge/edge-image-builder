package combustion

import "github.com/suse-edge/edge-image-builder/pkg/image"

func ComponentHelmCharts(ctx *image.Context) []image.HelmChart {
	if ctx.ImageDefinition.Kubernetes.Version == "" {
		return nil
	}

	var charts []image.HelmChart

	if ctx.ImageDefinition.Kubernetes.Network.APIVIP != "" {
		metalLBChart := image.HelmChart{
			Name:                  "metallb",
			Repo:                  "https://suse-edge.github.io/charts",
			TargetNamespace:       "metallb-system",
			CreateNamespace:       true,
			InstallationNamespace: "kube-system",
			Version:               "0.14.3",
		}

		endpointCopierOperatorChart := image.HelmChart{
			Name:                  "endpoint-copier-operator",
			Repo:                  "https://suse-edge.github.io/charts",
			TargetNamespace:       "endpoint-copier-operator",
			CreateNamespace:       true,
			InstallationNamespace: "kube-system",
			Version:               "0.2.0",
		}

		charts = append(charts, metalLBChart, endpointCopierOperatorChart)
	}

	return charts
}
