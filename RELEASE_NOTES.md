# Edge Image Builder Releases

# Next

## General

* Added output at combustion phase to observe the script being executed
* Kubernetes install scripts are now downloaded at runtime instead of during the container image build process
* Bumped Go Version to 1.22

## API

### Image Definition Changes

* Added the ability to configure Helm charts under `kubernetes/helmCharts`

### Image Configuration Directory Changes

* Helm chart values files can be specified under `kubernetes/helm/values`

## Bug Fixes

* [#239](https://github.com/suse-edge/edge-image-builder/issues/239) - Incorrect warning when checking for both .yml and .yaml files
* [#259](https://github.com/suse-edge/edge-image-builder/issues/259) - SCC registration is not cleaned up if RPM resolution fails
* [#260](https://github.com/suse-edge/edge-image-builder/issues/260) - Empty network directory produces a network configuration script
* [#267](https://github.com/suse-edge/edge-image-builder/issues/267) - Embedded registry renders Kubernetes resources even when Kubernetes is not configured
* [#242](https://github.com/suse-edge/edge-image-builder/issues/242) - Empty rpms directory triggers resolution

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
