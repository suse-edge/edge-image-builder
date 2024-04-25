# Installing Packages
This documentation dives deeper into how a user can configure packages for installation inside the EIB image. Furthermore, it explains the **RPM resolution** process that EIB goes through so it can ensure that configured packages will be successfully installed even in an **air-gapped** environment.

## Supported systems
EIB's **RPM resolution** process and package installation has been tested on the following `x86_64` systems: 
1. [SLES 15-SP5](https://www.suse.com/download/sles/)
1. [openSUSE Tumbleweed](https://get.opensuse.org/tumbleweed/)
1. [Ubuntu 22.04](https://releases.ubuntu.com/jammy/)
1. [Fedora Linux](https://fedoraproject.org/server/download)

## Specify packages for installation
You can configure packages for installation in the following ways:
1. Providing a `packageList` configuration under `operatingSystem.packages` in the EIB image configuration file
1. Creating a `rpms` directory under EIB's configuration directory and provide local RPM files that you want to be installed on the image

### Install packages through 'packageList'
To install a package using the `packageList` configuration, at a minimum you must configure the following under `operatingSystem.packages`:
1. Valid package names under `packageList`
1. Valid RPM repositories under `additionalRepos` or an SCC registration code under `sccRegistrationCode`

#### Install a package from a third-party repo
```yaml
operatingSystem:
  packages:
    packageList:
      - reiserfs-kmp-default-debuginfo
    additionalRepos:
      - url: https://download.opensuse.org/repositories/Kernel:/SLE15-SP5/pool
      - url: https://rpm.rancher.io/rke2/stable/common/slemicro/noarch
        unsigned: true
```
> **_NOTE:_** Before adding any repositories under `additionalRepos`, make sure that they are signed with a valid GPG key.
> **All non-signed additional repositories will cause EIB to fail unless they are explicitly labeled as `unsigned`.**

#### Install a package from SUSE's internal repositories
```yaml
operatingSystem:
  packages:
    packageList:
      - wget2
    sccRegistrationCode: <your-reg-code>
```

### Side-load RPMs
Sometimes you may want to install RPM files that are not hosted in a repository. For this use-case, you should create the following set of directories under EIB's configuration directory:

* `rpms` - Place your RPMs here. All RPMs in this directory will be checked for valid GPG signatures, included in the built image and installed during the combustion phase. 
  > **_NOTE:_** You must provide an `additionalRepos` entry or a `sccRegistrationCode` in your EIB definition file if your RPMs are dependent on other packages.
* `rpms/gpg-keys` - Place all GPG keys that are used to sign your RPMs here. All GPG keys in this directory will be used when validating the GPG signatures of your RPMs. **Trying to install RPMs that are unsigned or have unrecognized GPG keys will result in a failure of the EIB build process.**

If you want to install an unsigned RPM, refer to the [Installing unsigned packages](#installing-unsigned-packages) section of this documentation.

#### RPM with dependency resolution from a third-party repository  
EIB configuration directory tree:
```shell
.
├── eib-config-iso.yaml
├── base-images
│   └── SLE-Micro.x86_64-5.5.0-Default-RT-GM.raw
└── rpms
    ├── gpg-keys
    │   └── reiserfs-kpm-default-debuginfo.key
    └── reiserfs-kmp-default-debuginfo-5.14.21-150500.205.1.g8725a95.x86_64.rpm
```

EIB config file `packages` configuration:
```yaml
operatingSystem:
  packages:
    additionalRepos:
      - url: https://download.opensuse.org/repositories/Kernel:/SLE15-SP5/pool
```

#### RPM with dependency resolution from SUSE's internal repositories
EIB configuration directory tree:
```shell
.
├── eib-config-iso.yaml
├── base-images
│   └── SLE-Micro.x86_64-5.5.0-Default-RT-GM.raw
└── rpms
    ├── gpg-keys
    │   └── git.key
    └── git-2.35.3-150300.10.33.1.x86_64.rpm
```

EIB config file `packages` configuration:
```yaml
operatingSystem:
  packages:
    sccRegistrationCode: <your-reg-code>
```

### Installing unsigned packages
By default, EIB does GPG validation for every **additional repository** and **side-loaded RPM**.
If you wish to use unsigned additional repositories and/or unsigned RPMs you must add the `noGPGCheck: true` property to EIB's `packages` configuration, like so:
```yaml
operatingSystem:
  packages:
    noGPGCheck: true
```
By providing this configuration, **all** GPG validation will be **disabled**, allowing you to use non-signed packages.

> **_NOTE:_** This property is intended for development purposes only. For production use-cases we encourage users to always use EIB's GPG validation.