# Edge Image Builder Releases

# Next

## General

* Added support for prime/upstream kubernetes artifacts using the `config/artifacts.yaml` file
* Added mounting for `/usr/local` for Operating System file handling

## API

### Image Definition Changes

### Image Configuration Directory Changes

## Bug Fixes

---

# v1.2.1

## General

* Improved validation and handling for the image definition API Version

---

# v1.2.0

## General

* Added single-stack IPv6 and dual-stack networking support for Kubernetes
* SUSEConnect now properly activates the SL Micro "Extras" module
* Improved validation for `operatingSystem.enableFIPS` flag
* Added the ability to build RAW Encrypted Images
* Improved Embedded Artifact Registry handling to no longer be memory bound
* Updated Embedded Artifact Registry documentation
* Improved Helm chart handling to allow deploying multiple Helm charts with the same chart name
* Added warning when running cross-architecture builds
* Added support for authenticated container registries
* Dependency upgrades
  * go.mod is now using Go 1.24 (upgraded from 1.22)
  * Virtual IP addresses are now served by MetalLB 0.1.0+up0.14.9 (upgraded from 0.14.9)
  * Embedded registry is now utilizing Hauler v1.2.1 (upgraded from v1.0.7)

## API

### Image Definition Changes

* The current version of the image definition has been incremented to `1.2` to include the changes below 
  * Existing definitions using the `1.0` and `1.1` versions of the schema will continue to work with EIB
* Added `kubernetes.network.apiVIP6` field to enable cluster LoadBalancer based on IPv6 address
* Added `operatingSystem.enableExtras` flag to enable the SUSE Linux Extras repository during RPM resolution
* Added `operatingSystem.rawConfiguration.luksKey` field for specifying the LINUX UNIFIED KEY SETUP for modifying RAW Encrypted images
* Added `operatingSystem.rawConfiguration.expandEncryptedPartition` field to specify if the LUKS encrypted partition should be expanded during build time
* Added `kubernetes.helm.charts.releaseName` field to allow for deploying multiple instances of the same Helm chart
* Added `embeddedArtifactRegistry.registries` field to allow providing credentials for authenticated registries

### Image Configuration Directory Changes

## Bug Fixes

* [#591](https://github.com/suse-edge/edge-image-builder/issues/591) - Allow additional module registration during package resolution
* [#593](https://github.com/suse-edge/edge-image-builder/issues/593) - OS files script should mount /var
* [#594](https://github.com/suse-edge/edge-image-builder/issues/594) - Package installation breaks package resolution if packages are already installed on the root OS
* [#632](https://github.com/suse-edge/edge-image-builder/issues/632) - Create the required Elemental Agent directory structure during Combustion
* [#625](https://github.com/suse-edge/edge-image-builder/issues/625) - Cache is stale for images tagged `:latest`
* [#632](https://github.com/suse-edge/edge-image-builder/issues/606) - Allow for duplicate Helm chart names
* [#699](https://github.com/suse-edge/edge-image-builder/issues/699) - SL Micro 6.0/6.1 images updated via KIWI fail to build due to a different checksum format

---

# v1.1.1

## Bug Fixes

* [#699](https://github.com/suse-edge/edge-image-builder/issues/699) - SL Micro 6.0 images updated via KIWI fail to build due to a different checksum format

---

# v1.1.0

## General

* Adds support for customizing SL Micro 6.0 base images (for SLE Micro 5.5 images, EIB 1.0.x must still be used)
* Added the ability to build aarch64 images on an aarch64 host machine
* Added the ability to automatically copy files into the built images filesystem (see Image Configuration Directory Changes below)
* Kubernetes manifests are now applied in a systemd service instead of using the `/manifests` directory 
* Helm chart installation backOffLimit changed from 1000(default) to 20
* Added Elemental configuration validation
* Dropped `-chart` suffix from installed Helm chart names
* Added caching for container images
* Added built image name output to build command 
* Leftover combustion artifacts are now removed on first boot
* OS files and user provided certificates now maintain original permissions when copied to the final image
* Dependency upgrades
  * "Phone Home" deployments are now utilizing Elemental v1.6 (upgraded from v1.4)
  * Embedded registry is now utilizing Hauler v1.0.7 (upgraded from v1.0.1)
  * Network customizations are now utilizing nmc v0.3.1 (upgraded from v0.3.0)

## API

### Image Definition Changes

* The current version of the image definition has been incremented to `1.1` to include the changes below 
  * Existing definitions using the `1.0` version of the schema will continue to work with EIB
* Introduced a dedicated FIPS mode option (`enableFIPS`) which will enable FIPS mode on the node
* Adds an optional `apiVersions` field under Helm charts

### Image Configuration Directory Changes

* An optional directory named `os-files` may be included to copy files into the resulting image's filesystem at runtime
* The `custom/files` directory may now include subdirectories, which will be maintained when copied to the image
* Elemental configuration now requires a registration code in order to install the necessary RPMs from the official sources
  * Alternatively, the necessary Elemental RPMs can be manually side-loaded instead

## Bug Fixes

* [#481](https://github.com/suse-edge/edge-image-builder/issues/481) - Certain Helm charts fail when templated without specified API Versions
* [#491](https://github.com/suse-edge/edge-image-builder/issues/491) - Large Helm manifests fail to install
* [#498](https://github.com/suse-edge/edge-image-builder/issues/498) - Fix kernelArgs issue with Leap Micro 6.0
* [#543](https://github.com/suse-edge/edge-image-builder/issues/543) - Kernel cmdline arguments aren't honoured in SL Micro 6.0 for SelfInstall ISO's
* [#550](https://github.com/suse-edge/edge-image-builder/issues/550) - PackageHub inclusion in RPM resolution silently errors on SLE Micro 6.0
* [#565](https://github.com/suse-edge/edge-image-builder/issues/565) - K3S SELinux uses an outdated package

---

# v1.0.2

## General

* Added the ability to consume both 512/4096 byte sector size disk input base-images
* Added the ability to leverage Elemental node reset for unmanaged operating systems
* Added version command and version marker on CRB images

## Bug Fixes

* [#429](https://github.com/suse-edge/edge-image-builder/issues/429) - Automatically set execute bit on scripts
* [#447](https://github.com/suse-edge/edge-image-builder/issues/447) - Support >512 byte sector size base images
* [#442](https://github.com/suse-edge/edge-image-builder/issues/442) - Only get images from specific Kubernetes objects

---

# v1.0.1

## Bug Fixes

* [#405](https://github.com/suse-edge/edge-image-builder/issues/405) - OCI registries are assumed to include the chart name

---

# v1.0.0

## General

* Added a progress bar showing the progress of pulling images into the embedded artifact registry
* Added annotations to Helm CRs

## Bug Fixes

* [#352](https://github.com/suse-edge/edge-image-builder/issues/352) - Resizing raw images results in dracut-pre-mount failure
* [#355](https://github.com/suse-edge/edge-image-builder/issues/355) - Helm fails getting charts stored in unauthenticated OCI registries
* [#359](https://github.com/suse-edge/edge-image-builder/issues/359) - Helm validation does not check if a chart uses an undefined repository
* [#362](https://github.com/suse-edge/edge-image-builder/issues/362) - Helm templating failure
* [#365](https://github.com/suse-edge/edge-image-builder/issues/365) - Unable to locate downloaded Helm charts
* [#374](https://github.com/suse-edge/edge-image-builder/issues/374) - Enable SELinux support for Kubernetes agents if servers enforce it
* [#381](https://github.com/suse-edge/edge-image-builder/issues/381) - Empty gpg-keys directory passes GPG enablement only to fail during the dependency resolution
* [#383](https://github.com/suse-edge/edge-image-builder/issues/383) - Criteria for validating the OS definition does not include RPM
* [#372](https://github.com/suse-edge/edge-image-builder/issues/372) - Empty certificates directory does not raise a build error but fails to boot the node
* [#371](https://github.com/suse-edge/edge-image-builder/issues/371) - EIB allows an SSH key to be set for a user when createHome is set to false
* [#384](https://github.com/suse-edge/edge-image-builder/issues/384) - Improve RPM validation
* [#392](https://github.com/suse-edge/edge-image-builder/issues/392) - Users script does not unmount /home
* [#364](https://github.com/suse-edge/edge-image-builder/issues/364) - Kubernetes component output is jumbled when downloading the installer
* [#361](https://github.com/suse-edge/edge-image-builder/issues/361) - Raw image build can fail silently due to lack of space

---

# v1.0.0-rc3

## API

### Image Definition Changes

* Removed the `operatingSystem/isoConfiguration/unattended` option

## Bug Fixes

* [#319](https://github.com/suse-edge/edge-image-builder/issues/319) - Combustion fails when combustion directory content is larger than half of the RAM of the system
* [#233](https://github.com/suse-edge/edge-image-builder/issues/233) - Use different Helm chart sources for development and production builds
* [#337](https://github.com/suse-edge/edge-image-builder/issues/337) - Re-running raw builds should remove the previous built image
* [#95](https://github.com/suse-edge/edge-image-builder/issues/95)   - Compressed images are not supported
* [#343](https://github.com/suse-edge/edge-image-builder/issues/343) - Embedded Artifact Registry is memory bound
* [#341](https://github.com/suse-edge/edge-image-builder/issues/341) - Make Elemental registry configurable for production builds
* [#258](https://github.com/suse-edge/edge-image-builder/issues/258) - Kubernetes installation doesn't work with DHCP given hostname

---

# v1.0.0-rc2

## General

* Added output at combustion phase to observe the script being executed
* Kubernetes install scripts are now downloaded at runtime instead of during the container image build process
* Bumped Go Version to 1.22
* Added support for using Helm charts from authenticated repositories/registries
* Added support for skipping Helm chart TLS verification and for using Helm charts from plain HTTP repositories/registries
* Added support for providing CA files to Helm resolver for TLS verification
* Added minor formatting improvements to the CLI output

## API

* The `--config-file` argument to the EIB CLI has been renamed to `--definition-file`.
* The `--build-dir` argument to the EIB CLI is now optional and defaults to `<config-dir>/_build`, creating it if it does not exist.
* The `--config-dir` argument to the EIB CLI is now optional and defaults to `/eib` which is the most common mounted container volume.
* New `validate` subcommand is introduced
* The `--validate` argument to the `build` subcommand is now removed

### Image Definition Changes

* Added the ability to configure Helm charts under `kubernetes/helm`

### Image Configuration Directory Changes

* Helm chart values files can be specified under `kubernetes/helm/values`

## Bug Fixes

* [#239](https://github.com/suse-edge/edge-image-builder/issues/239) - Incorrect warning when checking for both .yml and .yaml files
* [#259](https://github.com/suse-edge/edge-image-builder/issues/259) - SCC registration is not cleaned up if RPM resolution fails
* [#260](https://github.com/suse-edge/edge-image-builder/issues/260) - Empty network directory produces a network configuration script
* [#267](https://github.com/suse-edge/edge-image-builder/issues/267) - Embedded registry renders Kubernetes resources even when Kubernetes is not configured
* [#242](https://github.com/suse-edge/edge-image-builder/issues/242) - Empty rpms directory triggers resolution
* [#283](https://github.com/suse-edge/edge-image-builder/issues/283) - Definition file argument to EIB is incorrect
* [#245](https://github.com/suse-edge/edge-image-builder/issues/245) - Pass additional arguments to Helm resolver
* [#307](https://github.com/suse-edge/edge-image-builder/issues/307) - Helm chart parsing logic breaks if "---" is present in the chart's resources
* [#272](https://github.com/suse-edge/edge-image-builder/issues/272) - Custom files should keep their permissions
* [#209](https://github.com/suse-edge/edge-image-builder/issues/209) - Embedded artifact registry starting even when manifests don't have any images
* [#315](https://github.com/suse-edge/edge-image-builder/issues/315) - If Elemental fails to register during Combustion we drop to emergency shell
* [#321](https://github.com/suse-edge/edge-image-builder/issues/321) - Certain Helm charts fail when templated in the `default` namespace
* [#289](https://github.com/suse-edge/edge-image-builder/issues/289) - The services for RPM dependency resolution failed to start

---

# v1.0.0-rc1

## General

* Added support for deploying user-provided Helm charts
* Added support for custom network configuration scripts

## API

### Image Definition Changes

* Removed the `embeddedArtifactRegistry/images/supplyChainKey` attribute
* Changed `operatingSystem/users/sshKey` into `operatingSystem/users/sshKeys` and it is now a list instead of a single string
* Added the ability to configure operating system groups under `operatingSystem/groups`
* Added optional `primaryGroup` field for operating system users
* Added optional `secondaryGroups` field for operating system users
* Added optional `createHomeDir` field for operating system users
* Added optional `uid` field for operating system users

## Bug Fixes

* [#197](https://github.com/suse-edge/edge-image-builder/issues/197) - Consider using ENTRYPOINT instead of CMD
* [#213](https://github.com/suse-edge/edge-image-builder/issues/213) - zypper clean after zypper install
* [#216](https://github.com/suse-edge/edge-image-builder/issues/216) - Update the docs to reflect that systemd can be used for any kind of systemd unit, not just services
