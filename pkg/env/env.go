package env

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
	Elemental struct {
		RegisterRepository    string
		SystemAgentRepository string
	}
}
