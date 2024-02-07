# Installing packages
This documentation dives deeper into how a user can configure packages for installation inside the EIB image. Furthermore, it explains the **RPM resolution** process that EIB goes through so it can ensure that configured packages will be successfully installed even in an **air-gapped** environment.

## Supported systems
EIB's **RPM resolution** process and package installation has been tested on the following `x86_64` systems: 
1. [SLES 15-SP5](https://www.suse.com/download/sles/)
1. [openSUSE Tumbleweed](https://get.opensuse.org/tumbleweed/)
1. [Ubuntu 22.04](https://releases.ubuntu.com/jammy/)
1. [Fedora Linux](https://fedoraproject.org/server/download)

## Specify packages for installation
You can configure packages for installation in the following ways:
1. provide a `packageList` configuration under `operatingSystem.packages` in the EIB image configuration file
1. create a `rpms` directory under EIB's configuration directory and provide local RPM files that you want to be installed on the image

### Install packages through 'packageList'
To install a package using the `packageList` configuration, at a minimum you must configure the following under `operatingSystem.packages`:
1. valid package names under `packageList`
1. `additionalRepos` or `sccRegistrationCode`

#### Install a package from a third-party repo
```yaml
operatingSystem:
  packages:
    packageList:
      - reiserfs-kmp-default-debuginfo
    additionalRepos:
      - url: https://download.opensuse.org/repositories/Kernel:/SLE15-SP5/pool
```
> **_NOTE:_** Before adding any repositories under `additionalRepos`, make sure that they are signed with a valid GPG key. **All non-signed additional repositories will cause EIB to fail.**

#### Install a package from SUSE's internal repositories
```yaml
operatingSystem:
  packages:
    packageList:
      - wget2
    sccRegistrationCode: <your-reg-code>
```

### Side-load RPMs
Sometimes you may want to install RPM files that are not hosted in a repository. For this use-case, you should create a directory called `rpms` under `<eib-config-dir>/rpms` and copy your local RPM files there.

> **_NOTE:_** You must provide an `additionalRepos` entry or a `sccRegistrationCode` if your RPMs are dependent on other packages.

All RPMs that will be side-loaded must have **valid** GPG signatures. The GPG keys used to sign the RPMs must be copied to a directory called `gpg-keys` which must be created under `<eib-config-dir>/rpms`. **Trying to install RPMs that are unsgined or have unrecognized GPG keys will result in a failure of the EIB build process.**

#### RPM with dependency resolution from a third-party repository  
EIB configuration directory tree:
```shell
.
├── eib-config-iso.yaml
├── images
│   └── SLE-Micro.x86_64-5.5.0-Default-RT-GM.raw
└── rpms
    ├── gpg-keys
    │   └── reiserfs-kpm-default-debuginfo.key
    └── reiserfs-kmp-default-debuginfo-5.14.21-150500.205.1.g8725a95.x86_64.rpm
```

EIB config file `packages` configuration:
```yaml
operatingSystem:
  packages:
    sccRegistrationCode:
      - url: https://download.opensuse.org/repositories/Kernel:/SLE15-SP5/pool
```

#### RPM with depdendency resolution from SUSE's internal repositories
EIB configuration directory tree:
```shell
.
├── eib-config-iso.yaml
├── images
│   └── SLE-Micro.x86_64-5.5.0-Default-RT-GM.raw
└── rpms
    ├── gpg-keys
    │   └── git.key
    └── git-2.35.3-150300.10.33.1.x86_64.rpm
```

EIB config file `packages` configuration:
```yaml
operatingSystem:
  packages:
    additionalRepos: <your-reg-code>
```

### Installing unsigned packages
By default EIB does GPG validation for every **additional repository** and **side-loaded RPM**. If you wish to use unsigned additional repositories and/or unsinged RPMs you must add the `noGPGCheck: true` property to EIB's `packages` configuration, like so:
```yaml
operatingSystem:
  packages:
    noGPGCheck: true
```
By providing this configuration, **all** GPG validation will be **disabled**, allowing you to use non-signed pacakges.

> **_NOTE:_** This property is intended for development purposes only. For production use-cases we encourage users to always use EIB's GPG validation.

## Package installation workflow
The package installation workflow can be separated in three logical parts:
1. *Running the EIB container* - how to run the EIB container so that the **RPM resolution** has the needed permissions
1. *Building the EIB image* - what happens during the **RPM resolution** logic of EIB's image build
1. *Booting the EIB image* - how are the packages actually installed once the EIB image is booted for the first time

### Running the EIB container
![image](./images/rpm-eib-container-run.png)

The package installation workflow begins with the user configuring packages and/or stand-alone RPMs that will be installed when the EIB image is booted. On how to to do a correct configuration, see [Configure packages for installation](#configure-packages-for-installation).

After the desired configuration has been made, the user runs the EIB container with the [`--privileged`](https://docs.podman.io/en/latest/markdown/podman-run.1.html#privileged) option, ensuring that EIB has the needed permissions to successfully run a Podman instance within its container. This is a crutial prerequisite for building a working EIB image with package installation configured (more on this in the next section). 

An example of the command can be seen below:
```shell
podman run --rm --privileged -it \
-v $IMAGE_DIR:/eib eib:dev /bin/eib \
-config-file $CONFIG_FILE.yaml \
-config-dir /eib \
-build-dir /eib/_build
```

> **_NOTE:_** Depending on the `cgroupVersion` that Podman operates with, you might also need to run the command with `root` permissions. This is the case for `cgroupVersion: v1`. In this version, non-root usage of the `--privileged` option is not supported. For `cgroupVersion: v2`, non-root usage is supported. 
>
>In order to check the `cgroupVersion` that Podman operates with, run the following command:
>```shell
>podman info | grep cgroupVersion
>```

Once the EIB container has been successfully executed, it parses all the user provided configuration and begins the **RPM resolution** process. 

### Building the EIB image
During this phase, EIB prepares the user configured packages for installation. This process is called **RPM resolution** and it includes:
1. Validating that each provided package has a GPG signature or comes from a GPG signed RPM repository
1. Resolving and downloading the dependencies for each configured package
1. Creating a RPM repository that consists of the configured packages and their dependencies
1. Configure the usage of this repositry for package installation during the **combustion** phase of the EIB image boot

#### RPM resolution process
![image](./images/rpm-resolver-architecture.png)

EIB mainly utilizes Podman's functionality to setup the environment needed for the **RPM resolution** process. In order to communicate with Podman, EIB first creates a [listening service](https://docs.podman.io/en/latest/markdown/podman-system-service.1.html) that will faciliate the communication between EIB and Podman. From here onwards, asume that any Podman related operation that EIB does goes through the **listening service** first.

Once EIB establishes communication with Podman, it parses the user configured ISO/RAW file and converts it to a Podman importable **virtual disk tarball**. This tarball is [imported](https://docs.podman.io/en/stable/markdown/podman-import.1.html) as an image in Podman. 

EIB then proceeds to build the **RPM resolver** image using the **tarball image** as a base. This procedure ensures that the validation/resolution of any packages that are configured for installation will be as close to the desired user environment as possible.

All the RPM resolution logic is done during the build of the **RPM resolver** image. This includes, but is not limited to:
1. Environment setup:
    * Connecting to SUSE's internal RPM repositories, if configured by the user through `operatingSystem.packages.sccRegistrationCode`
    * Importing any GPG keys provided by the user under `<eib-config-dir>/rpms/gpg-keys`
    * Adding any third-party RPM repositories, if configured by the user through `operatingSystem.packages.additionalRepos`
1. Validation of:
    * RPM files provided by the user under `<eib-config-dir>/rpms`
    * Third-party RPM repositories, if configured by the user through `operatingSystem.packages.additionalRepos`
1. Downloading the dependencies for all configured packages and side-loaded RPMs to a **RPM cache directory**

After a successful RPM resolver image build, EIB starts a container from the newly built image and copies the aforementioned **RPM cache directory** to the **combustion** directory located in the EIB container. This cache directory is then converted to a ready to use RPM repository by EIB.

The final step in the EIB **RPM resolution** process is to create an **install script** which uses the aforementioned RPM repository to install the user configured packages during the EIB image combustion phase.

#### Troubleshooting
When troubleshooting the **RPM resolution** process, it is beneficial to look at the following files/directories inside of the EIB build directory:
1. `eib-build.log` - general logs for the whole EIB image build process
1. `podman-image-build.log` - logs for the build of the EIB resolver image. If missing, but the `resolver-image-build` directory is present, this means that there is a problem in the configuration of the `resolver-image-build` directory
1. `podman-system-service.log` - logs for the Podman listening service
1. `resolver-image-build` directory - build context for the resolver image. Make sure that the `Dockerfile` holds correct data. When installing side-loaded RPMs, make sure that the `rpms` and `gpg-keys` directories are present in the `resolver-image-build` directory
1. `resolver-base-image` direcotry - contains resources related to the creation of the **virtual disk tarball** archive. If this directory exists, this means that a problem has been encountered while EIB was trying to import the **tarball image**
1. `prepare-base.log` - logs related to the creation of the **virtual disk tarball**
1. `createrepo.log` - logs related to the conversion of the **RPM cache directory** to a **RPM repository**
1. `combustion/rpm-repo` directory - the **RPM repository**; should hold the desired RPMs for installation and their dependencies
1. `combustion/10-rpm-install.sh` - script that will be executed during the **combustion** phase; should use the `rpm-repo` repository and have all the expected packages configured for installation

### Booting the EIB image
During the combustion phase of the EIB image boot, as mentioned above, both the **RPM repository** and **RPM combustion script** will be present in the combustion [configuration directory](https://github.com/openSUSE/combustion?tab=readme-ov-file#combustion) respectively under `/dev/shm/combustion/config/10-rpm-install.sh` and `/dev/shm/combustion/config/rpm-repo`.

The root combustion script then calls the `10-rpm-install.sh` script, which does the following:
1. Adds the `rpm-repo` directory as a local RPM repository for its package manager
1. Installs the desired packages from the newly added `rpm-repo` repository 
1. Once all packages have been installed it removes the `rpm-repo` from the package manager

The successful execution of the `10-rpm-install.sh` script indicates that all packages have been installed on the operating system. Upon the completion of the image boot, the user should have access to every package that he configured when building the EIB image.