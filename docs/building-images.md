# Building Images

Two things are necessary to build an image using EIB:
1. A definition file that describes the image to build
1. A directory that contains the base SLE Micro image to modify, along with any other custom files that
   will be included in the built image

# Image Definition File

The Image Definition File is a YAML document describing a single image to build. The file is specified using
the `--definition-file` argument. Only a single image may be built at a time, however the same image configuration
directory may be used to build multiple images by creating multiple definition files.

> **_NOTE:_** Unless otherwise specified, all sections and fields are optional.

## Required Fields

The following can be used as the minimum configuration required to create an image. Each field in this section is
required for each image definition.

```yaml
apiVersion: 1.0
image:
  imageType: iso
  arch: x86_64
  baseImage: SLE-Micro.x86_64-5.5.0-Default-SelfInstall-GM.install.iso
  outputImageName: eib-image.iso
```

* `apiVersion` - Indicates the version of the definition file schema for EIB to expect.
* `imageType` - Must be either `iso` or `raw` depending on the type of image being customized.
* `arch` - Must be `x86_64`; future versions of EIB will support multiple architectures.
* `baseImage` - Indicates the name of the image file used as the base for the built image. Base image files must be
  uncompressed before they can be modified by EIB. This file must be located
  under the `base-images` directory of the image configuration directory (see below for more information).
  The image will **not** directly be modified by EIB; a new image will be created each time EIB is run.
* `outputImageName` - Indicates the name of the image that EIB will build. This may only be a filename; the image will
  be written to the root of the image configuration directory.

## Operating System

The operating system configuration section is entirely optional and should not be included unless one or more
customizations are being applied.

The following describes the possible options for the operating system section:
```yaml
operatingSystem:
  <TYPE SPECIFIC CONFIGURATION (see below)>
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
      - url: https://example1.com
      - url: https://example2.com
        unsigned: true
    sccRegistrationCode: scc-reg-code
```

### Type-specific Configuration

Depending on the type of image being customized, one of the following optional sections may be included.

* `isoConfiguration` - Optional; configuration in this section only applies to ISO images.
  * `installDevice` - Optional; specifies the disk that should be used as the install
  device. This needs to be block special, and will default to automatically wipe any data found on the disk.
  Additionally, specifying this attribute triggers a GRUB override to automatically install the operating
  system rather than prompting user to begin the installation, allowing for a fully unattended and automated
  installation. If omitted, the user will be prompted to select the "Install" option from the GRUB menu, 
  as well as having to select the installation disk and confirm that the device
  will be wiped in the process.
* `rawConfiguration` - Optional; configuration in this section only applies to RAW images.
  * `diskSize` - Optional; sets the desired raw disk image size that EIB will resize the resulting image to.
  This is important to ensure that your disk image is large enough to accommodate any artifacts being embedded
  in the image. It is advised to set this to slightly smaller than your SD card size (or block device if writing
  directly to a disk) as the system will automatically expand at boot time to fill the size of the block device.
  This is optional, but highly recommended. Specify as an integer with either "M" (Megabyte), "G" (Gigabyte),
  or "T" (Terabyte) as a suffix (e.g. "32G").

### General

The remainder of the operating system customizations may be applied regardless of image type. 

* `time` - Defines timezone information and NTP configuration.
  * `timezone` - Specifies the timezone in the format of "Region/Locality" (e.g. "Europe/London").
  The full list may be found by running `timedatectl list-timezones` on a Linux system.
  * `ntp` - Defines attributes related to configuring NTP.
    * `forceWait` - Requests that NTP attempts to synchronize timesources before starting other services,
    with a 180s timeout.
    * `pools` - Specifies a list of pools that NTP will use as data sources.
    * `servers` - Specifies a list of servers that NTP will use as data sources.
* `proxy` - Defines system-wide proxy information.
  * `httpProxy` - Sets the system-wide http proxy settings.
  * `httpsProxy` - Sets the system-wide https proxy settings.
  * `noProxy` - Overrides the default `NO_PROXY` list. By default, this is `localhost, 127.0.0.1` if this
  parameter is omitted. If this option is set, the default entries will need to be manually added if they are
  still in use.
* `kernelArgs` - Provides a list of flags that should be passed to the kernel on boot.
* `groups` - Defines a list of operating system groups to create. This will not fail if the 
group already exists. Each entry is made up of the following fields:
  * `name` - Required; Name of the group to create.
  * `gid` - Optional; If specified, the group will be created with the given ID. If omitted, the GID will be generated
    by the operating system.
* `users` - Defines a list of operating system users to create. Each entry is made up of
  the following fields (one or both of the password and SSH key must be provided per user):
  * `username` - Required; Username of the user to create. To set the password or SSH key for the root user,
    use the value `root` for this field.
  * `uid` - If specified, the user will be created with the given ID. If omitted, the UID will be generated
    by the operating system.
  * `createHomeDir` - If set to `true`, a home directory will be created for the user. Defaults to `false`
  if unspecified. If one or more SSH keys is specified, this must be set to `true` to properly configure the
  user.
  * `encryptedPassword` - Encrypted password to set for the use (for example,
  using `openssl passwd -6 $PASSWORD` to generate the value for this field).
  * `sshKeys` - List of public SSH keys to configure for the user.
  * `primaryGroup` - If specified, the user will be configured with this value as the primary group. The group
  must already exist, either as a default group or one defined in the `groups` field. If this is omitted, the
  result will be the default for the operating system (on SLE Micro, this is `users`).
  * `secondaryGroups` - If specified, the user will be configured as part of each listed group. The
  groups must already exist, either as default groups or as ones defined in the `groups` field.
* `systemd` - Defines lists of systemd units to enable/disable. Either or both of `enable` and `disable` may
be included; if neither are provided, this section is ignored.
  * `enable` - Defines a list of systemd services to enable.
  * `disable` - Defines a list of systemd services to disable.
* `keymap` - Sets the virtual console (VC) keymap. The full list of options may be found by running
`localectl list-keymaps` on a Linux system. If unset, EIB will default this value to `us`.
* `packages` - Defines packages that will be installed when the node is booted. EIB will determine the necessary
dependencies and download them into the built image. For detailed information on how to use this configuration,
see the [Installing pacakges](.installing-packages.md) guide.
  * `noGPGCheck` - Defines if GPG validation should be disabled for all additional repositories and side-loaded
  RPMs. **Disabling GPG validation is intended for development purposes only.**
  * `packageList` - Defines a list of packages to install from SUSE's internal RPM repositories or
  from additionally provided third-party repositories.
  * `additionalRepos` - Defines a list of third-party RPM repositories that will be added to the package manager of
  the node. Each entry is made up of the following:
    * `url` - Required; Specifies the URL of the repository.
    * `unsigned` - This must be set to `true` if the repository is unsigned. 
  * `sccRegistrationCode` - Specifies the SUSE Customer Center registration code in plain text, which is used to
  connect to SUSE's internal RPM repositories.

## Kubernetes

The Kubernetes configuration section is entirely optional and should not be included unless one or more
customizations are being applied.

This section contains all necessary settings to configure and bootstrap a Kubernetes cluster using either K3s or RKE2.

> **_NOTE:_** In addition to the configuration below, if you are building a `raw` image, you must manually specify its
> disk size. The disk size specification is needed in order to ensure that the raw image has enough space to host
> the Kubernetes tarball resources that EIB copies into it. Increasing the raw image disk size is done in the
> [`rawConfiguration`](#operating-system) property.

```yaml
kubernetes:
  version: v1.28.8+rke2r1
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

* `version` - Required; Specifies the version of a particular K3s or RKE2 release (e.g.`v1.28.8+k3s1` or `v1.28.8+rke2r1`)
* `network` - Required for multi-node clusters, optional for single-node clusters; Defines the network configuration 
for bootstrapping a cluster.
  * `apiVIP` - Required for multi-node clusters, optional for single-node clusters; Specifies the IP address which
  will serve as the cluster LoadBalancer, backed by MetalLB.
  * `apiHost` - Optional; Specifies the domain address for accessing the cluster.
* `nodes` - Required for multi-node clusters; Defines a list of all nodes that form the cluster.
  * `hostname` - Required; Indicates the fully qualified domain name (FQDN) to identify the particular node on which
  the remainder of these attributes will be applied.
  * `type` - Required; Selects the Kubernetes node type, either `server` (for control plane nodes) or
  `agent` (for worker nodes).
  * `initializer` - Optional; Indicates which node should function as the cluster initializer. The initializer node is
  the server node which bootstraps the cluster and allows other nodes to join it. If unset, the first server in the
  node list will be selected as the initializer.
* `manifests` - Defines a list of manifests that will be applied to the cluster automatically when it starts.
  Can be used separately or in combination with the configuration directory.
  * `urls` - Specifies the list of HTTP(s) URLs to download the manifests from. These are downloaded at build time and
  will be included in the built image.
* `helm` - Defines a set of Helm charts to be deployed to the cluster. The charts and associated images are downloaded
at build time and included in the built image.
  * `charts` - Required; Defines a list of Helm charts and configuration for each Helm chart.
    * `name` - Required; This must match the name of the actual Helm chart.
    * `repositoryName` - Required; Specifies which repository within the `repositories` section contains this
    Helm chart. This must match the `name` attribute on one of the repositories defined in the next section.
    * `version` - Required; The version of the Helm chart to be deployed.
    * `installationNamespace` - Optional; The namespace where the Helm installation is executed. If omitted,
    the default is `default`.
    * `targetNamespace` - Optional; The namespace where the Helm chart will be deployed. If omitted, the default
    is `default`.
    * `createNamespace` - Optional; If `true` the `targetNamespace` will be created. If `false`, it assumes the
    `targetNamespace` already exists. If `false` and the namespace doesn't exist, the deployment will fail at boot time.
    * `valuesFile` - Optional; The name of the [Helm values file](https://helm.sh/docs/chart_template_guide/values_files/)
    (not including the path) that will be applied to this chart. The values file must be placed under
    `kubernetes/helm/values` for the specified chart.
  * `repositories` - Required if one or more chart is specified; Defines a list of Helm repositories/registries
  required for each chart.
    * `name` - Required; Defines the name for this repository. This name doesn't have to match the name of the actual
    repository, but must correspond with the `repositoryName` of one or more charts.
    * `url` - Required; Defines the URL which contains the Helm repository containing a chart or the OCI registry
    URL to a chart.
    * `caFile` - Optional; The name of the CA File (not including the path), placed under `kubernetes/helm/certs`, for
    the specified repository/registry.
    * `plainHTTP` - Optional; Must be set to `true` when connecting to repositories and registries over plain HTTP.
    * `skipTLSVerify` - Optional; Must be set to `true` for repositories and registries with untrusted TLS certificates.
    * `authentication` - Required for authenticated repositories/registries.
      * `username` - Required; Defines the username for accessing the specified repository/registry. 
      * `password` - Required; Defines the password for accessing the specified repository/registry.

## SUSE Manager (SUMA)

The SUMA configuration section is entirely optional and should not be included unless one or more
customizations are being applied.

The image may be configured to automatically register with SUSE Manager at boot time. If this is enabled, 
the system will need a valid network configuration, either via DHCP or configured statically. For air-gapped images,
the registration server must be available within the air-gapped network for this to work.

> **_NOTE_**: If the activation key is not in the root organization, the organization ID must be included as a prefix, for example "2-yourkey".
> as a prefix to the key itself (e.g `11-slemicro55` instead of simply `slemicro55`)

The following describes the possible options for the SUMA section:

```yaml
suma:
  host: suma.edge.suse.com
  activationKey: slmicro55
```

The following items **must** be defined in the configuration file under the `suma` section:

* `host` - Specifies the FQDN of the SUSE Manager host to register against. This must only be the FQDN for the
server; the prefix (HTTP, HTTPS) should not be specified.
* `activationKey` - Specifies the activation key that the node uses to register.

Additionally, the appropriate `venv-salt-minion` RPM package must be supplied in the RPM directory
(see the [RPM side-loading documentation](#rpms) for more information). This RPM is required at boot time prior
to SUSE Manager registration taking place. This RPM can usually be found on the
SUSE Manager host itself at `https://<suma-host>/pub/repositories/slemicro/5/5/bootstrap/x86_64/`.

## Embedded Artifact Registry

The embedded artifact registry configuration section is entirely optional and should not be included unless one or more
customizations are being applied.

This section defines an internal registry to be deployed on the resulting node. This registry hosts all container
images used by manifests and Helm charts for deploying workloads at boot time. The embedded artifact registry will
be automatically deployed if images are detected in user provided manifests or Helm charts, even if it is
not explicitly configured in this section.

The following describes the possible options for the embedded artifact registry section:

```yaml
embeddedArtifactRegistry:
  images:
    - name: hello-world:latest
    - name: ghcr.io/fluxcd/flux-cli@sha256:02aa820c3a9c57d67208afcfc4bce9661658c17d15940aea369da259d2b976dd
```

* `images` - Defines a list of container images to download and host on the node.
  * `name` - Required; Specifies the name, with a tag or digest, of a container image to be pulled and stored.

# Image Configuration Directory

The Image Configuration Directory contains all the files necessary for EIB to build an image.

## Required Files & Directories

```shell
.
├── definition-1.yaml
├── definition-2.yaml
└── base-images
    ├── SLE-Micro.x86_64-5.5.0-Default-SelfInstall-GM2.install.iso
    └── SLE-Micro.x86_64-5.5.0-Default-GM.raw
```

* `*.yaml` - All image definition files should be in the root of the image configuration directory. Multiple definition
files may be included in a single configuration directory, with the specific definition file specified as a CLI argument.
* `base-images` - This directory must exist and contains the base images from which EIB will build customized images.
There are no restrictions on the naming of the image files themselves. The image definition file will specify the name
of the image in this directory to use for a particular build.

## Certificates 

Certificate files stored in this directory will be installed on the node when it boots.

```shell
.
├── definition.yaml
└── certificates
    ├── my-ca.pem
    └── my-ca.crt
```

* `certificates` - If present, all files with the extension ".pem" or ".crt" will be installed as CA certificates
in the built image.

## RPMs

The [Operating System](#operating-system) section of the image definition defines RPMs to install from hosted 
repositories. Alternatively, RPM files may be included in the image configuration directory. These RPMs are
bundled in the image and installed at boot in the same way as RPMs specified in image definition. More information
on the details of this process can be found in the [Side-load RPMs](installing-packages.md#side-load-rpms)
section of the [Installing Packages](installing-packages.md) guide.

```shell
.
├── definition.yaml
└── rpms
    ├── my-policy.rpm
    └── gpg-keys
        └── my-key.gpg
```

* `rpms` - If present, one or more RPMs must be included in this directory. 
  * `gpg-keys` - Contains the GPG keys, if any, used to validate the RPMs in the parent directory.

## Network Configuration

The network configuration for multiple nodes may be specified in a single image. For more information on the format
of these files, see the [nm-configurator](https://github.com/suse-edge/nm-configurator/) documentation. 

```bash
.
├── definition.yaml
└── network
    ├── node1.suse.com.yaml
    └── node2.suse.com.yaml
```

* `network` - May be included to inject custom network configuration script or desired network configurations.
  * `configure-network.sh` - If present, this script will be used to initialize the network during the combustion phase.
  Otherwise, network configurations will be generated from all desired states in this directory and will be included
  in the built image. The configurations relevant for the particular host will be identified and applied during
  the combustion phase.

## Kubernetes

In addition to the [Kubernetes configuration in the image definition](#kubernetes), additional files may be added
to the image configuration directory for inclusion in the built image. The structure and use of these files is
defined by the Kubernetes cluster being installed.

```shell
.
├── definition.yaml
└── kubernetes
    ├── config
    │   ├── agent.yaml
    │   └── server.yaml
    └── manifests
        └── my-manifest.yaml.yaml
```

* `kubernetes` - May be included to inject cluster specific configurations, apply manifests, and install Helm charts.
  * `config` - Contains [K3s](https://docs.k3s.io/installation/configuration#configuration-file) or
  [RKE2](https://docs.rke2.io/install/configuration#configuration-file) cluster configuration files that will be
  applied to the provisioned Kubernetes cluster.
    * `server.yaml` - If present, this configuration file will be applied to all control plane nodes.
    * `agent.yaml` - If present, this configuration file will be applied to all worker nodes.
  * `manifests` - Contains locally provided manifests which will be applied to the cluster. Can be used separately or
    in combination with the manifests section in the definition file. All files in this directory will be parsed and
    the container images that they reference will be downloaded and served in an embedded artefact registry.
  * `helm` - Contains locally provided Helm charts and value files which will be applied to the cluster.
    * `values` - Contains [Helm values files](https://helm.sh/docs/chart_template_guide/values_files/). Helm charts
    that require specified values must have a values file included in this directory.
    * `certs` - Contains certificate files/bundles for TLS verification. Untrusted HTTPS-enabled Helm repositories and
    registries must be provided with a certificate file/bundle or require `skipTLSVerify` to be true.

## Elemental

Automatic Elemental registration may be configured for the image. The Elemental registration configuration file,
which can be downloaded using the Elemental extension in Rancher, must be placed in this configuration directory.

```bash
.
├── definition.yaml
└── elemental
    └── elemental_config.yaml
```

* `elemental` - This must contain a file named `elemental_config.yaml`. This file will be bundled in
the built image and used to register with Elemental on boot.

> **_NOTE:_** Elemental builds use EIB's package resolution process to download any necessary RPM packages. 
> To ensure a successful build, this process requires the ```--privileged``` flag to be passed to the
> ```podman run``` command. For more info on why this is required, please see
> [Package resolution design](design/pkg-resolution.md#running-the-eib-container).

## Custom

EIB has the ability to bundle in custom scripts that will be run during the combustion phase when a node is
booted with the built image. Additionally, custom files may be included, however they are not automatically
deployed on the node when it boots. If these files are needed beyond the combustion phase, a script should
be included that explicitly copies them to the filesystem.

Combustion scripts are executed alphabetically. All scripts that EIB automatically includes will be prefixed using
values between 00 and 49 (e.g. `05-configure-network.sh`, `30-suma-register.sh`). Unless absolutely sure the default
flow should be interrupted, all custom scripts should be prefixed within the range 50-99 (e.g. `60-my-script.sh`) or
not begin with a number.

```bash
.
├── definition.yaml
└── custom
    ├── files
    │   ├── custom-binary
    │   └── custom-script.sh
    └── scripts
        └── 70-manual-configuration.sh
```

* `custom` - May be included to inject files into the built image. Files are organized by subdirectory as follows:
  * `scripts` - If present, all the files in this directory will be included in the built image and automatically
    executed during the combustion phase.
  * `files` - If present, all the files in this directory will be available at combustion time on the booted node.
