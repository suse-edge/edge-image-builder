# Edge Image Builder Releases

# Next

## General

## API

### Image Definition Changes

### Image Configuration Directory Changes

## Bug Fixes
* [#239](https://github.com/suse-edge/edge-image-builder/issues/239) - Incorrect warning when checking for both .yml and .yaml files

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
