# Building Images

Two things are necessary to build an image using EIB:
1. A definition file that describes the image to build
1. A directory that contains the base SLE Micro image to modify, along with any other custom files that
   will be included in the built image

## Image Definition File

The Image Definition File is a YAML document describing a single image to build. The file is specified using
the `-config-file` argument. Only a single image may be built at a time, however the same image configuration
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
* `baseImage` - Indicates the name of the image file used as the base for the built image. This file must be located
  under the `images` directory of the image configuration directory (see below for more information). This image will
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
    unattended: false
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
  If left omitted, the user will still have to select the disk to install to (if >1 found) and confirm wipe.
  * `unattended` - Optional; forces GRUB override to automatically install the operating
  system rather than prompting user to begin the installation. In combination with `installDevice` can create
  a fully unattended and automated install. Beware of creating boot loops and data loss with these options.
  If left omitted (or set to `false`) the user will still have to choose to install via the GRUB menu.
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
* `systemd` - Optional; Defines lists of services to enable/disable. Either or both of `enable` and `disable` may
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

## SUSE Manager (SUMA)

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

## Embedded Artifact Registry

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

### RPMs

Custom RPMs may be included in the configuration directory. These RPMs will be bundled into the built image
and installed when the image is booted. The following describes the directory structure needed to configure this:

* `rpms` - All RPMs in this directory will be included in the built image and installed during the
  combustion phase. These RPMs are installed directly (instead of using zypper), which means that there will be no
  automatic dependency resolution.

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
