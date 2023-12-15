#  Template Fields
#  BaseImage   - image to use as base for the build of this image
#  FromRPMPath - path to the custom RPM directory relative to the resolver image build context in the EIB container
#  ToRPMPath   - path to the custom RPM directory relative to the resolver image
#  RegCode     - scc.suse.com registration code
#  AddRepo     - additional third-party repositories that will be used in the resolution process
#  CacheDir    - zypper cache directory where all rpm dependencies will be downloaded to
#  PkgList     - list of packages for which to do the dependency resolution
FROM {{.BaseImage}}

{{ if and (ne .FromRPMPath "") (ne .ToRPMPath "") -}}
COPY {{ .FromRPMPath }} {{ .ToRPMPath -}}
{{ end }}

{{ if ne .RegCode "" -}}
RUN suseconnect -r {{.RegCode}}
RUN SLE_SP=$(cat /etc/rpm/macros.sle | awk '/sle/ {print $2};' | cut -c4) && suseconnect -p PackageHub/15.$SLE_SP/x86_64
RUN zypper ref
{{ end }}

{{ if ne .AddRepo "" -}}
RUN counter=1 && \
    for i in {{.AddRepo}}; \
    do \
      zypper ar --no-gpgcheck -f $i addrepo$counter; \
      counter=$((counter+1)); \
    done
{{ end }}

RUN zypper \
    --pkg-cache-dir {{.CacheDir}} \ 
    --no-gpg-checks \
    install -y \
    --download-only \
    --force-resolution \
    --auto-agree-with-licenses \
    --allow-vendor-change \
    -n {{.PkgList}}

RUN touch {{.CacheDir}}/zypper-success

{{ if ne .RegCode "" -}}
RUN suseconnect -d
{{ end }}

# ensure that when a container is started
# enought time will be given for the copy 
# command to copy the pkg-cache-dir
CMD [ "sleep", "60m" ]