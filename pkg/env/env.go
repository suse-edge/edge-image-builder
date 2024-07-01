package env

var (
	ElementalPackageRepository = "https://download.opensuse.org/repositories/isv:/Rancher:/Elemental:/Maintenance:/5.5/standard/"
)

type ArtifactSources struct {
	MetalLB struct {
		Chart      string
		Repository string
		Version    string
	}
	EndpointCopierOperator struct {
		Chart      string
		Repository string
		Version    string
	}
}
