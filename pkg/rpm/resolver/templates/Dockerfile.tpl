#  Template Fields
#  BaseImage    - image to use as base for the build of this image
#  FromRPMPath  - path to the custom RPM directory relative to the resolver image build context in the EIB container
#  ToRPMPath    - path to the custom RPM directory relative to the resolver image
#  FromGPGPath  - path to the directory holding the GPG keys for the custom RPMs relative to the resolver image build context in the EIB container
#  ToGPGPath    - path to the directory holding the GPG keys for the custom RPMs relative to the resolver image
#  RegCode      - scc.suse.com registration code
#  AddRepo      - additional third-party repositories that will be used in the resolution process
#  CacheDir     - zypper cache directory where all rpm dependencies will be downloaded to
#  PKGList      - list of packages for which to do the dependency resolution
#  LocalRPMList - list of local RPMs for which dependency resolution has to be done
#  LocalGPGList - list of local GPG keys that will be imported in the resolver image
#  NoGPGCheck   - when set to true skips the GPG validation for all third-party repositories and local RPMs
FROM {{ .BaseImage }}

{{ if and .FromRPMPath .ToRPMPath -}}
COPY {{ .FromRPMPath }} {{ .ToRPMPath }}
{{ if and .FromGPGPath .ToGPGPath -}}
COPY {{ .FromGPGPath }} {{ .ToGPGPath }}
{{ end -}}
{{ end -}}

{{ if ne .RegCode "" }}
RUN suseconnect -r {{ .RegCode }}
RUN SLE_SP=$(cat /etc/rpm/macros.sle | awk '/sle/ {print $2};' | cut -c4) && suseconnect -p PackageHub/15.$SLE_SP/x86_64
RUN zypper ref
{{ end -}}

{{- range $index, $repo := .AddRepo }}

{{- $gpgCheck := "" -}}
{{- if $.NoGPGCheck -}}
{{ $gpgCheck = "--no-gpgcheck" }}
{{- else if .Unsigned -}}
{{ $gpgCheck = "--gpgcheck-allow-unsigned-repo" }}
{{- end -}}

RUN zypper ar {{ $gpgCheck }} -f {{ .URL }} addrepo {{- $index }}

{{ end -}}

{{ if and .LocalGPGList (not .NoGPGCheck) }}
RUN rpm --import {{ .LocalGPGList }}
{{ end -}}

{{ if and .LocalRPMList (not .NoGPGCheck) }}
RUN rpm -Kv {{ .LocalRPMList }}
{{ end -}}

RUN zypper \
    --pkg-cache-dir {{.CacheDir}} \ 
    --gpg-auto-import-keys \
    {{ if .NoGPGCheck -}}
    --no-gpg-checks \
    {{ end -}}
    install -y \
    --download-only \
    --force-resolution \
    --auto-agree-with-licenses \
    --allow-vendor-change \
    -n {{.PKGList}} {{.LocalRPMList}}

RUN touch {{.CacheDir}}/zypper-success

{{ if ne .RegCode "" }}
RUN suseconnect -d
{{ end -}}

CMD ["/bin/bash"]