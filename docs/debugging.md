# Debugging

# Build Directory

By default, EIB will create a `_build` directory inside of the image configuration directory. This behavior can
be overridden using the `--build-dir` flag to set a new directory. Keep in mind that this directory is relative to
inside the EIB container itself. If the new value is not mounted into the container through `podman`, the 
logs and cached files will be deleted when the container process ends.

Within the build directory, a new directory will be created for each build. This directory will be named according
to a timestamp (e.g. `build-Mar13_18-46-51`). Each individual build directory contains:

* All log files related to the build itself
* The exact contents of the combustion directory that is included in the RTD image

Additionally, there may be a `cache` directory under the build directory (`_build/cache` by default). This directory
contains files downloaded by EIB during build time, such as the RKE2 installer bits. If this directory is present
when EIB performs a build that uses any of these files, they will be pulled from the cache instead of downloading again.

# Log Files

The following describes the possible log files that will be found in the directory for each individual build.

## General

### `eib-build.log`

Primary log file for the overall EIB build. This should be the first place to check when errors are encountered.

### `raw-build.log`

Log for the process EIB uses to modify a raw image file. These modifications include injecting the combustion directory
and optionally resizing the image if the definition indicates to.

### `iso-extract.log`

Before an ISO image can be modified by EIB, the contents of it need to be extracted. This log file tracks the
process of both extracting the contents of the ISO and unsquashing the embedded raw image so it too
can be modified.

### `iso-build.log`

Log for the process EIB uses to build a modified self-installing ISO. This log will include the results of resquashing
the raw image embedded in the ISO and setting the unattended and install device configurations.

## Networking

### `network-config.log`

Log file containing the results of generating network configurations with [nm-configurator](https://github.com/suse-edge/nm-configurator/).

## RPM Dependency Resolution

The [design document](./design/pkg-resolution.md) describes in detail the process used to calculate and download all
of the RPMs necessary for an air-gapped installation. The log files below are used at various steps during the process.

### `podman-image-build.log`

Log for the build of the container image that EIB will run to perform the dependency resolution. 

What does it mean if the `podman-image-build.log` is missing? This usually means that there has been a problem in either:
1. The import of the base virtual disk tarball image that the EIB resolver uses
1. The configuration of the resolver-image-build directory; examples include the files failed to copy to the resolver
   container image build context, or the Dockerfile was not rendered properly

Types of errors this log could produce:
* Any Dockerfile context errors - missing files/directories error, missing base image error, etc.
* Errors related to `suseconnect` registration/de-registration
* Errors related to adding third-party repositories to `zypper`
* Errors related to validating the GPG signatures of any side-loaded RPMs, third-party repositories,
  or packages scheduled for installation
* Any package installation errors - missing package or package dependencies error for example

### `podman-system-service.log`

Log for the podman listening service that EIB uses to communicate with the podman instance running inside of the EIB
container.

Types of errors this log could produce:
* Errors related to any podman operations that EIB does
* Errors related to the podman `/run/podman/podman.sock` file

### `prepare-resolver-base-tarball-image.log`

Logs related to the creation of the virtual disk tarball that EIB uses as base for its resolver image.

Types of errors this log could produce:
* Errors related to manipulating the ISO file so that EIB can get its raw virtual machine disk image
* Errors related to packaging the virtual machine disk image into a tarball 

### `createrepo.log`

Logs related to the conversion of the RPM cache directory to an RPM repository.

Types of errors this log could produce:
* Errors related to the `createrepo` command (https://linux.die.net/man/8/createrepo).

## Embedded Artifact Registry

### `embedded-registry.log`

Logs related to the creation and population of the embedded artifact registry that contains all of the container
images that need to be available on the RTD image.

Types of errors this log could produce:
* Errors related to not being able to find a user-specified container image
* Errors related to not being able to access a specified container image (e.g. Docker registry may sometimes block
  you from pulling a container image if you pull it too many times)

## Helm

### `helm-repo-add.log`

Logs related to adding a Helm repository to the EIB Helm resolver.

Types of messages this log could produce:
* Warning related to the repository having been added (doesn't cause any errors)
* Errors related to not being able to find a user-specified repository
* Errors related to the user-specified URL not being a valid repository
* Errors related to the repository giving an HTTP response when it was expecting an HTTPs response
  (a possible solution is to set `plainHTTP` to `true`)
* Errors related to the repository not being secure/passing TLS verification
  (a possible solution is to set `skipTLSVerify` to `true` or provide the CA cert file for TLS verification)
* Errors related to not being authorized to access the repository
  (a possible solution is to add a username and password for authenticated repositories)

### `helm-registry-login.log`

Logs related to logging into a Helm registry by the EIB Helm resolver.

Types of messages this log could produce:
* Errors related to not being able to find the user-specified registry
* Errors related to the user-specified URL not being a valid registry
* Errors related to the registry giving an HTTP response when it was expecting an HTTPs response
  (a possible solution is to set `plainHTTP` to `true`)
* Errors related to the registry not being secure/passing TLS verification
  (a possible solution is to set `skipTLSVerify` to `true` or provide the CA cert file for TLS verification)
* Errors related to not being authorized to access the registry
  (a possible solution is to add a username and password for authenticated repositories)

### `helm-pull.log`

Logs related to downloading a Helm chart by the EIB Helm resolver.

Types of messages this log could produce:
* Errors related to not being able to find the Helm chart (for example, an incorrectly specified version or the wrong
  name of the Helm chart was given)
* Errors related to the URL not being a valid Helm chart
* Errors related to the registry giving an HTTP response when it was expecting an HTTPs response
  (a possible solution is to set `plainHTTP` to `true`)
* Errors related to the registry not being secure/passing TLS verification
  (a possible solution is to set `skipTLSVerify` to `true` or provide the CA cert file for TLS verification)
* Errors related to not being authorized to access the registry
  (a possible solution is to add a username and password for authenticated repositories)

### `helm-template.log`

Logs related to templating a Helm chart by the EIB Helm resolver.

Types of messages this log could produce:
* Will show the output of the `helm template` command on a Helm chart using the provided values
