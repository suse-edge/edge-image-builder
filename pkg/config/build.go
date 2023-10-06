package config

type BuildConfig struct {
	ImageOutputDir  string
	BuildTempDir    string
	DeleteArtifacts bool

	CombustionDir     string
	CombustionScripts []string
}

func (bc *BuildConfig) AddCombustionScript(filename string) {
	bc.CombustionScripts = append(bc.CombustionScripts, filename)
}