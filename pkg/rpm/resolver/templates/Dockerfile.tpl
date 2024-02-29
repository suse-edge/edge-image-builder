#  Template Fields
#  BaseImage               - image to use as base for the build of this image
#  FromRPMPath             - path to the custom RPM directory relative to the resolver image build context in the EIB container
#  ToRPMPath               - path to the custom RPM directory relative to the resolver image
#  FromGPGPath             - path to the directory holding the GPG keys for the custom RPMs relative to the resolver image build context in the EIB container
#  ToGPGPath               - path to the directory holding the GPG keys for the custom RPMs relative to the resolver image
#  RPMResolutionScriptName - name of the RPM resolution script
FROM {{ .BaseImage }}

COPY {{ .RPMResolutionScriptName }} {{ .RPMResolutionScriptName }}

{{ if and .FromRPMPath .ToRPMPath -}}
COPY {{ .FromRPMPath }} {{ .ToRPMPath }}
{{ if and .FromGPGPath .ToGPGPath -}}
COPY {{ .FromGPGPath }} {{ .ToGPGPath }}
{{ end -}}
{{ end }}

RUN ./{{ .RPMResolutionScriptName }}

CMD ["/bin/bash"]