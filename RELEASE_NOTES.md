# Edge Image Builder Releases

# Next

## General

## API

### Image Definition Changes

### Image Configuration Directory Changes

## Bug Fixes

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
