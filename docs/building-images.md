# Building Images

Two things are necessary to build an image using EIB:
1. A definition file that describes the image to build
1. A directory that contains the base SLE Micro image to modify, along with any other custom files that
   will be included in the built image

## Image Definition File

The Image Definition File is a YAML document describing a single image to build. The file is specified using
the `--definition-file` argument. Only a single image may be built at a time, however the same image configuration
directory may be used to build multiple images by creating multiple definition files.

The following can be used as the minimum configuration required to create an image:
```yaml
apiVersion: 1.0
image:
  imageType: iso
  arch: x86_64
  baseImage: SLE-Micro.x86_64-5.5.0-Default-SelfInstall-GM.install.iso
  outputImageName: eib-image.iso
```

* `apiVersion` - Indicates the version of the definition file schema for EIB to expect
* `imageType` - Must be either `iso` or `raw`.
* `arch` - Must be either `x86_64` or `aarch64`.
* `baseImage` - Indicates the name of the image file used as the base for the built image. Base image files must be uncompressed. This file must be located
  under the `base-images` directory of the image configuration directory (see below for more information). This image will
  **not** directly be modified by EIB; a new image will be created each time EIB is run.
* `outputImageName` - Indicates the name of the image that EIB will build. This may only be a filename; the image will
  be written to the root of the image configuration directory.

### Operating System

The operating system configuration section is entirely optional.

The following describes the possible options for the operating system section:
```yaml
operatingSystem:
  isoConfiguration:
    installDevice: /path/to/disk
  time:
    timezone: Europe/London
    ntp:
      forceWait: true
      pools:
        - 1.pool.server.com
      servers:
        - 10.0.0.1
        - 10.0.0.2
  proxy:
    httpProxy: http://10.0.0.1:3128
    httpsProxy: http://10.0.0.1:3128
    noProxy:
    - localhost
    - 127.0.0.1
    - edge.suse.com
  kernelArgs:
  - arg1
  - arg2
  groups:
    - name: group1
    - name: group2
  users:
  - username: user1
    encryptedPassword: 123
    sshKeys:
      - user1Key1
      - user1Key2
    primaryGroup: groupPrimary
    secondaryGroups:
      - group1
      - group2
  - username: user2
    encryptedPassword: 456
    secondaryGroups:
      - group3
  - username: user3
    sshKeys:
      - user3Key
  systemd:
    enable:
      - service0
      - service1
    disable:
      - serviceX
  keymap: us
  packages:
    noGPGCheck: false
    packageList:
      - pkg1
      - pkg2
    additionalRepos:
      - url: https://foo.bar
      - url: https://foo.baz
        unsigned: true
    sccRegistrationCode: scc-reg-code
```

* `isoConfiguration` - Optional; configuration in this section only applies to ISO images.
  * `installDevice` - Optional; specifies the disk that should be used as the install
  device. This needs to be block special, and will default to automatically wipe any data found on the disk.
  Additionally, forces GRUB override to automatically install the operating
  system rather than prompting user to begin the installation. Allowing for
  a fully unattended and automated install. Beware of creating boot loops and data loss with this option.
  If left omitted, the user will still have to select the disk to install to (if >1 found) and confirm wipe as well as 
  choose to install via the GRUB menu.
* `rawConfiguration` - Optional; configuration in this section only applies to RAW images only.
  * `diskSize` - Optional; sets the desired raw disk image size. This is important to ensure that your disk
  image is large enough to accommodate any artifacts that you're embedding. It's advised to set this to slightly
  smaller than your SDcard size (or block device if writing directly to a disk) and the system will automatically
  expand at boot time to fill the size of the block device. This is optional, but highly recommended. Specify in
  integer format with either "M" (Megabyte), "G" (Gigabyte), or "T" (Terabyte) as a suffix, e.g. "32G".
* `time` - Optional; section where the user can provide timezone information and Chronyd configuration.
  * `timezone` - Optional; the timezone in the format of "Region/Locality", e.g. "Europe/London". Full list via `timedatectl list-timezones`.
  * `ntp` - Optional; contains attributes related to configuring NTP
    * `forceWait` - Optional; requests that Chrony attempts to synchronize timesources before starting other services (with a 180s timeout).
    * `pools` - Optional; a list of pools that Chrony can use as data sources.
    * `servers` - Optional; a list of servers that Chrony can use as data sources.
* `proxy` - Optional; section where the user can provide system-wide proxy information
  * `httpProxy` - Optional; set the system-wide http proxy settings
  * `httpsProxy` - Optional; set the system-wide https proxy settings
  * `noProxy` - Optional; override the default `NO_PROXY` list. By default, this is "localhost, 127.0.0.1" if this
  parameter is omitted. If this option is set, these may need to be manually added if they are still in use.
* `kernelArgs` - Optional; Provides a list of flags that should be passed to the kernel on boot.
* `groups` - Optional; Defines a list of operating system groups to be created. This will not fail if the
  group already exists. Each entry is made up of the following fields:
  * `name` - Required; Name of the group to create.
  * `gid` - Optional; If specified, the group will be created with the given ID. If omitted, the GID will be generated
    by the operating system.
* `users` - Optional; Defines a list of operating system users to be created. Each entry is made up of
  the following fields (one or both of the password and SSH key must be provided per user):
  * `username` - Required; Username of the user to create. To set the password or SSH key for the root user,
    use the value `root` for this field.
  * `uid` - Optional; If specified, the user will be created with the given ID. If omitted, the UID will be generated
    by the operating system.
  * `createHomeDir` - Optional; If set to `true`, a home directory will be created for the user. Defaults to `false`
    if unspecified.
  * `encryptedPassword` - Optional; Encrypted password to set for the use (for example, using `openssl passwd -6 $PASSWORD`
    to generate the value for this field).
  * `sshKeys` - Optional; List of public SSH keys to configure for the user.
  * `primaryGroup` - Optional; If specified, the user will be configured with this as the primary group. The group
    must already exist, either as a default group or one defined in the `groups` field. If this is omitted, the
    result will be the default for the operating system (on SLE Micro, this is `users`).
  * `secondaryGroups` - Optional; If specified, the user will be configured as part of each listed group. The
    groups must already exist, either as default groups or as ones defined in the `groups` field.
* `systemd` - Optional; Defines lists of systemd units to enable/disable. Either or both of `enable` and `disable` may
  be included; if neither are provided, this section is ignored.
  * `enable` - Optional; List of systemd services to enable.
  * `disable` - Optional; List of systemd services to disable.
* `keymap` - Optional; sets the virtual console (VC) keymap, full list via `localectl list-keymaps`. If unset, we default to
  `us`.
* `packages` - Optional; Defines packages that should have their dependencies determined and pre-loaded into the built image. For detailed information on how to use this configuration, see the [Installing pacakges](installing-packages.md) guide.
  * `noGPGCheck` - Optional; Defines whether GPG validation should be disabled for all additional repositories and side-loaded RPMs. **Disabling GPG validation is intended for development purposes only!**
  * `packageList` - Optional; List of packages that are to be installed from SUSE's internal RPM repositories or from additionally provided third-party repositories.
  * `additionalRepos` - Optional; List of third-party RPM repositories that will be added to the package manager of the OS.
  * `sccRegistrationCode` - Optional; SUSE Customer Center registration code, used to connect to SUSE's internal RPM repositories.

### Kubernetes

The Kubernetes configuration section is another entirely optional one.
It contains all necessary settings to configure and bootstrap a Kubernetes cluster.
The supported Kubernetes distributions are K3s and RKE2.

> **_NOTE:_** In addition to the configuration below, if you are building a `raw` image, you must manually specify its disk size. The disk size specification is needed in order to ensure that the `raw` image has enough space to host the Kubernetes tarball resources that EIB attempts to copy into it. Increasing the `raw` image disk size is done through the [`rawConfiguration`](#operating-system) property.

```yaml
kubernetes:
  version: v1.28.0+rke2r1
  network:
    apiVIP: 192.168.122.100
    apiHost: api.cluster01.hosted.on.edge.suse.com
  nodes:
    - hostname: node1.suse.com
      type: server
    - hostname: node2.suse.com
      type: server
      initializer: true
    - hostname: node3.suse.com
      type: agent
    - hostname: node4.suse.com
      type: server
    - hostname: node5.suse.com
      type: agent
  manifests:
    urls:
      - https://k8s.io/examples/application/nginx-app.yaml
  helm:
    charts:
      - name: metallb
        version: 0.14.3
        repositoryName: suse-edge
        valuesFile: metallb-values.yaml
        targetNamespace: metallb-system
        createNamespace: true
        installationNamespace: kube-system
      - name: kubevirt
        version: 0.2.2
        repositoryName: suse-edge
      - name: apache
        version: 10.7.0
        repositoryName: apache-repo
    repositories:
      - name: suse-edge
        url: https://suse-edge.github.io/charts
      - name: apache-repo
        url: oci://registry-1.docker.io/bitnamicharts
        plainHTTP: false
        skipTLSVerify: true
        authentication:
          username: user
          password: pass
```

* `version` - Required; version string of a particular K3s or RKE2 release e.g.`v1.28.0+k3s1` or `v1.28.0+rke2r1`
* `network` - Required for multi-node clusters, optional for single-node clusters; network configuration for bootstrapping a cluster
  * `apiVIP` - Required for multi-node clusters, optional for single-node clusters; IP address which will serve as the cluster LoadBalancer (backed by MetalLB)
  * `apiHost` - Optional; domain address for accessing the cluster
* `nodes` - Required for multi-node clusters; list of all nodes forming the cluster
  * `hostname` - Required; a fully qualified domain name (FQDN) which identifies the particular node
  * `type` - Required; Kubernetes node type - either `server` (for control plane nodes) or `agent` (for worker nodes)
  * `initializer` - Optional; specifies the cluster initializer. The initializer node is the server node which bootstraps the cluster
     and allows other nodes to join it. If unset, the first server in the node list will be selected as the initializer.
* `manifests` - Optional; manifests to apply to the cluster.
  Can be used separately or in combination with the [configuration directory](#kubernetes-1).
  * `urls` - Optional; list of HTTP(s) URLs to download the manifests from
* `helm` - Optional; Helm charts to be deployed to the cluster.
  * `charts` - Required; Defines a list of Helm charts and configuration for each Helm chart.
    * `name` - Required; This must match the name of the actual Helm chart.
    * `repositoryName` - Required; This is the name of the corresponding `name` for a repository/registry specified within `repositories` that contains this Helm chart.
    * `version` - Required; The version of the Helm chart to be deployed.
    * `installationNamespace` - Optional; The namespace where the Helm installation is executed. The default is `default`.
    * `targetNamespace` - Optional; The namespace where the Helm chart will be deployed. The default is `default`.
    * `createNamespace` - Optional; If `true` the `targetNamespace` will be created, if `false` it assumes the `targetNamespace` already exists. If `false` and the namespace doesn't exist, the deployment will fail at boot time.
    * `valuesFile` - Optional; The name of the [Helm values file](https://helm.sh/docs/chart_template_guide/values_files/) (not including the path), placed under `kubernetes/helm/values`, for the specified chart (e.g. the input for `kubernetes/helm/values/metallb-values.yaml` is  `metallb-values.yaml`).
  * `repositories` - Required for charts; Defines a list of Helm repositories/registries required for each chart.
    * `name` - Required; Defines the name for this repository. This name doesn't have to match the name of the actual repository, but must correspond with the `repositoryName` of one or more charts.
    * `url` - Required; Defines the URL which contains the Helm repository containing a chart, or the OCI registry URL to a chart.
    * `caFile` - Optional; The name of the CA File (not including the path), placed under `kubernetes/helm/certs`, for the specified repository/registry (e.g. the input for `kubernetes/helm/certs/helm.crt` is  `helm.crt`).
    * `plainHTTP` - Optional; Must be set to `true` when connecting to repositories and registries over plain HTTP.
    * `skipTLSVerify` - Optional; Must be set to `true` for repositories and registries with untrusted TLS certificates.
    * `authentication` - Required for authenticated repositories/registries.
      * `username` - Required; Defines the username for accessing the specified repository/registry. 
      * `password` - Required; Defines the password for accessing the specified repository/registry.

### SUSE Manager (SUMA)

Automatic SUSE Manager registration can be configured for the image, which will happen at system-boot time. Therefore,
your system will need to come up with networking, either via DHCP or configured statically, e.g. via `nmc` or via
custom scripts. If you're creating an *air-gapped* image, do *not* use the SUSE Manager registration unless your server
is available from within the air-gapped network.

The following items must be defined in the configuration file under the `suma` section:

* `host` - This is the FQDN of the SUSE Manager host that the host needs to register against (do not use http/s prefix)
* `activationKey` - This is the activation key that the node uses to register with.

The default SSL certificate for the SUSE Manager server can usually be found at
`https://<suma-host>/pub/RHN-ORG-TRUSTED-SSL-CERT`.

Additionally, the appropriate *venv-salt-minion* RPM package must be supplied in the RPM's directory so it can be
installed at boot time prior to SUSE Manager registration taking place. This RPM can usually be found on the
SUSE Manager host itself at `https://<suma-host>/pub/repositories/slemicro/5/5/bootstrap/x86_64/` as an example.

### Embedded Artifact Registry

The embedded artifact registry configuration section is entirely optional. This is an internal registry that hosts all container images manually specified, as well as all container images automatically detected within user provided Helm charts and manifests.

The embedded artifact registry will be automatically deployed if images are detected within user provided manifests or Helm charts, even if it is not manually configured.

The following describes the possible options for the embedded artifact registry section:
```yaml
embeddedArtifactRegistry:
  images:
    - name: hello-world:latest
    - name: ghcr.io/fluxcd/flux-cli@sha256:02aa820c3a9c57d67208afcfc4bce9661658c17d15940aea369da259d2b976dd
```

* `images` - Configuration in this section only applies to container images.
  * `name` - required; specifies the name, with a tag or digest, of a container image to be pulled and stored.

## Image Configuration Directory

The Image Configuration Directory contains all the files necessary for EIB to build an image. As the project matures,
the structure of this directory will be better fleshed out. For now, the required structure is described below:

```shell
.
├── eib-config-iso.yaml
├── eib-config-raw.yaml
└── base-images
    └── SLE-Micro.x86_64-5.5.0-Default-SelfInstall-GM.install.iso
    └── SLE-Micro.x86_64-5.5.0-Default-GM.raw
```

* `eib-config-iso.yaml`, `eib-config-raw.yaml` - All image definition files should be in the root of the image
  configuration directory. Multiple definition files may be included in a single configuration directory, with
  the specific definition file specified as a CLI argument as described above.
* `base-images` - This directory must exist and contains the base images from which EIB will build customized images.
  There are no restrictions on the naming. The image definition file will specify which image in this directory
  to use for a particular build.

There are a number of optional directories that may be included in the image configuration directory:

* `certificates` - If present, all files with the extension ".pem" or ".crt" will be installed as CA certificates
  in the built image.
* `custom` - May be included to inject files into the built image. Files are organized by subdirectory as follows:
  * `scripts` - If present, all the files in this directory will be included in the built image and automatically
    executed during the combustion phase. Combustion scripts are executed alphabetically. All scripts that EIB
    automatically includes will be prefixed using values between 00 and 49 (e.g. `05-configure-network.sh`,
    `30-suma-register.sh`). Unless absolutely sure the default flow should be interrupted, all custom scripts
    should be prefixed within the range 50-99 (e.g. `60-my-script.sh`).
  * `files` - If present, all the files in this directory will be included in the built image.
* `network` - May be included to inject custom network configuration script or desired network configurations.
  * `configure-network.sh` - If present, this script will be used to initialize the network during the combustion phase.
  Otherwise, network configurations will be generated from all desired states in this directory
  and will be included in the built image. The configurations relevant for the particular host will be identified
  and applied during the combustion phase. Check [nm-configurator](https://github.com/suse-edge/nm-configurator/)
  for more information.

The following sections further describe optional directories that may be included.

### Kubernetes

* `kubernetes` - May be included to inject cluster specific configurations, apply manifests and install Helm charts.
  * `config` - Contains [K3s](https://docs.k3s.io/installation/configuration#configuration-file) or
    [RKE2](https://docs.rke2.io/install/configuration#configuration-file) cluster configuration files
    * `server.yaml` - If present, this configuration file will be applied to all control plane nodes
    * `agent.yaml` - If present, this configuration file will be applied to all worker nodes
  * `manifests` - Contains locally provided manifests which will be applied to the cluster. Can be used separately or
    in combination with the manifests section in the definition file. All files in this directory will be parsed and
    the container images that they reference will be downloaded and served in an embedded artefact registry.
  * `helm`
    * `values` - Contains [Helm values files](https://helm.sh/docs/chart_template_guide/values_files/). Helm charts that require specified values must have a values file.
    * `certs` - Contains cert files/bundles for TLS verification. Untrusted HTTPS-enabled Helm repositories and registries need to be provided a cert file/bundle or require `skipTLSVerify` to be true.

> **_NOTE:_** Image builds enabling SELinux mode in the configuration files use EIB's package resolution process
> to download any necessary RPM packages. To ensure a successful build, this process requires the ```--privileged```
> flag to be passed to the ```podman run``` command. For more info on why this is required, please see
> [Package resolution design](design/pkg-resolution.md#running-the-eib-container).

### RPMs

Custom RPMs may be included in the configuration directory. For more information on how to add custom RPMs, see the [Side-load RPMs](installing-packages.md#side-load-rpms) section of the [Installing packages](installing-packages.md) guide.

### Elemental

Automatic Elemental registration may be configured for the image. The Elemental registration configuration file,
which can be downloaded using the Elemental extension in Rancher, must be placed in the configuration directory
as follows:

* `elemental` - This must contain a file named `elemental_config.yaml`. This file will be bundled in
  the built image and used to register with Elemental on boot.

> **_NOTE:_** Elemental builds use EIB's package resolution process to download any necessary RPM packages. 
> To ensure a successful build, this process requires the ```--privileged``` flag to be passed to the
> ```podman run``` command. For more info on why this is required, please see
> [Package resolution design](design/pkg-resolution.md#running-the-eib-container).
