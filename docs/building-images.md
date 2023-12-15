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
  baseImage: SLE-Micro.x86_64-5.5.0-Default-SelfInstall-GM.install.iso
  outputImageName: eib-image.iso
```

* `apiVersion` - Indicates the version of the definition file schema for EIB to expect
* `imageType` - Must be either `iso` or `raw`.
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
  kernelArgs:
  - arg1
  - arg2
  users:
  - username: user1
    password: 123
    sshKey: user1Key
  - username: user2
    password: 456
  - username: user3
    sshKey: user3Key
  systemd:
    enable:
      - service0
      - service1
    disable:
      - serviceX
```

* `kernelArgs` - Optional; Provides a list of flags that should be passed to the kernel on boot.
* `users` - Optional; Defines a list of operating system users to be created. Each entry is made up of
  the following fields:
  * `username` - Required; Username of the user to create. To set the password or SSH key for the root user,
    use the value `root` for this field.
  * `password` - Optional; Encrypted password to set for the use (for example, using `openssl passwd -6 $PASSWORD`
    to generate the value for this field).
  * `sshKey` - Optional; Full public SSH key to configure for the user.
* `systemd` - Optional; Defines lists of services to enable/disable. Either or both of `enable` and `disable` may
  be included; if neither are provided, this section is ignored.
  * `enable` - Optional; List of systemd services to enable.
  * `disable` - Optional; List of systemd services to disable.

## SUSE Manager (SUMA)

Automatic SUSE Manager registration can be configured for the image, which will happen at system-boot time. Therefore
your system will need to come up with networking, either via DHCP or configured statically, e.g. via `nmc` or via
custom scripts. If you're creating an *air-gapped* image, do *not* use the SUSE Manager registration unless your server
is available from within the air-gapped network.

The following items must be defined in the configuration file under the `suma` section:

* `host` - This is the FQDN of the SUSE Manager host that the host needs to register against (do not use http/s prefix)
* `activationKey` - This is the activation key that the node uses to register with.
* `getSSL` - This specifies whether EIB should download and install the SUMA SSL Certificate (default: false)

The default SSL certificate for the SUSE Manager server can usually be found at
`https://<suma-host>/pub/RHN-ORG-TRUSTED-SSL-CERT`, and is currently hardcoded to look at this location relative to `host`.

Additionally, the appropriate *venv-salt-minion* RPM package must be supplied in the RPM's directory so it can be
installed at boot time prior to SUSE Manager registration taking place. This RPM can usually be found on the
SUSE Manager host itself at `https://<suma-host>/pub/repositories/slemicro/5/5/bootstrap/x86_64/` as an example.

## Image Configuration Directory

The Image Configuration Directory contains all the files necessary for EIB to build an image. As the project matures,
the structure of this directory will be better fleshed out. For now, the required structure is described below:

```shell
.
├── eib-config-iso.yaml
├── eib-config-raw.yaml
└── images
    └── SLE-Micro.x86_64-5.5.0-Default-SelfInstall-GM.install.iso
    └── SLE-Micro.x86_64-5.5.0-Default-GM.raw
```

* `eib-config-iso.yaml`, `eib-config-raw.yaml` - All image definition files should be in the root of the image
  configuration directory. Multiple definition files may be included in a single configuration directory, with
  the specific definition file specified as a CLI argument as described above.
* `images` - This directory must exist and contains the base images from which EIB will build customized images. There
  are no restrictions on the naming; the image definition file will specify which image in this directory to use
  for a particular build.

There are a number of optional directories that may be included in the image configuration directory:

* `network` - If present, network configurations will be generated from all desired states in this directory
  and will be included in the built image. The configurations relevant for the particular host will be identified
  and applied during the combustion phase. Check [nm-configurator](https://github.com/suse-edge/nm-configurator/)
  for more information.
* `custom` - May be included to inject files into the built image. Files are organized by subdirectory as follows:
  * `scripts` - If present, all the files in this directory will be included in the built image and automatically
    executed during the combustion phase.
  * `files` - If present, all the files in this directory will be included in the built image.

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

Additionally, the following RPMs must be included in the RPMs directory as described above:
* `elemental-register`
* `elemental-system-agent`
